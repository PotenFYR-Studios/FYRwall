package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

// Client is the server-side typed Unix-socket client. It implements
// firewall.FirewallBackend without giving the web process command execution.
type Client struct {
	path string
}

func NewClient(path string) *Client {
	if path == "" {
		path = SocketPath
	}
	return &Client{path: path}
}

func (c *Client) call(ctx context.Context, op string, params any, out any) error {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = b
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "unix", c.path)
	if err != nil {
		return fmt.Errorf("%w: agent socket: %v", firewall.ErrBackendUnavailable, err)
	}
	defer conn.Close()
	deadline := time.Now().Add(65 * time.Second)
	_ = conn.SetDeadline(deadline)
	if err := json.NewEncoder(conn).Encode(Request{Op: op, Params: raw}); err != nil {
		return err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("agent %s: %s", op, resp.Error)
	}
	if out != nil && len(resp.Payload) > 0 {
		return json.Unmarshal(resp.Payload, out)
	}
	return nil
}

type Capabilities struct {
	Backend    firewall.PolicyOwner     `json:"backend"`
	Ownership  firewall.OwnershipReport `json:"ownership"`
	Operations []string                 `json:"operations"`
	Protocol   int                      `json:"protocol"`
}

func (c *Client) Capabilities(ctx context.Context) (Capabilities, error) {
	var v Capabilities
	err := c.call(ctx, "GetCapabilities", nil, &v)
	return v, err
}

func (c *Client) OwnershipReport(ctx context.Context) (firewall.OwnershipReport, error) {
	capabilities, err := c.Capabilities(ctx)
	return capabilities.Ownership, err
}

func (c *Client) Name() string { return "agent" }
func (c *Client) Detect(ctx context.Context) firewall.DetectionResult {
	cap, err := c.Capabilities(ctx)
	return firewall.DetectionResult{Name: string(cap.Backend), Found: err == nil, Active: err == nil, Details: errorText(err)}
}
func (c *Client) Status(ctx context.Context) (v firewall.FirewallStatus, err error) {
	err = c.call(ctx, "GetFirewallStatus", nil, &v)
	return
}
func (c *Client) ListRules(ctx context.Context) ([]firewall.Rule, error) {
	var v struct {
		Rules []firewall.Rule `json:"rules"`
	}
	err := c.call(ctx, "ListRules", nil, &v)
	return v.Rules, err
}
func (c *Client) Validate(ctx context.Context, tx firewall.Transaction) (v firewall.ValidationResult) {
	if err := c.call(ctx, "ValidateTransaction", tx, &v); err != nil {
		v.Errors = []string{err.Error()}
	}
	return
}
func (c *Client) Snapshot(ctx context.Context, reason string) (v firewall.Snapshot, err error) {
	err = c.call(ctx, "CreateSnapshot", map[string]string{"reason": reason}, &v)
	return
}
func (c *Client) Apply(ctx context.Context, tx firewall.Transaction) (v firewall.ApplyResult, err error) {
	err = c.call(ctx, "ApplyTransaction", tx, &v)
	return
}
func (c *Client) Verify(ctx context.Context, expected firewall.StateHash) (v firewall.VerifyResult, err error) {
	err = c.call(ctx, "VerifyState", map[string]string{"expected": string(expected)}, &v)
	return
}
func (c *Client) Restore(ctx context.Context, snap firewall.Snapshot) error {
	return c.call(ctx, "RestoreSnapshot", snap, nil)
}
func (c *Client) Reload(ctx context.Context) error  { return c.call(ctx, "Reload", nil, nil) }
func (c *Client) Restart(ctx context.Context) error { return c.call(ctx, "Restart", nil, nil) }

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
