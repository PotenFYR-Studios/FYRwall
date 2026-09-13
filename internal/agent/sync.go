package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/system"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

type remoteIdentity struct {
	AgentID        string         `json:"agent_id"`
	Credential     string         `json:"credential"`
	Sequence       int64          `json:"sequence"`
	PolicyRevision int64          `json:"policy_revision"`
	PendingResult  *pendingResult `json:"pending_result,omitempty"`
	Certificate    string         `json:"certificate,omitempty"`
	PrivateKey     string         `json:"private_key,omitempty"`
}

type pendingResult struct {
	CommandID string   `json:"command_id"`
	Response  Response `json:"response"`
}

// RunRemoteSync maintains an outbound authenticated session using restart-safe
// long polling. No listener or inbound management port is opened on targets.
func (s *Server) RunRemoteSync(ctx context.Context, cfg *config.Config, log zerolog.Logger) error {
	if cfg.Agent.ServerURL == "" {
		return nil
	}
	id, err := loadAgentID(cfg.Agent.IDFile)
	if err != nil {
		return err
	}
	identity, err := loadRemoteIdentity(cfg.Agent.CredentialFile)
	if err != nil || identity.AgentID != id || identity.Credential == "" || (strings.HasPrefix(cfg.Agent.ServerURL, "https://") && identity.Certificate == "") {
		identity, err = enrollRemote(ctx, cfg, id)
		if err != nil {
			return err
		}
		if err = saveRemoteIdentity(cfg.Agent.CredentialFile, identity); err != nil {
			return err
		}
	}
	client, err := remoteHTTPClient(identity)
	if err != nil {
		return err
	}
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return nil
		}
		if certificateNeedsRotation(identity.Certificate, 7*24*time.Hour) {
			rotated, rotateErr := rotateRemoteCertificate(ctx, client, cfg.Agent.ServerURL, identity)
			if rotateErr == nil {
				identity.Certificate, identity.PrivateKey = rotated.Certificate, rotated.PrivateKey
				_ = saveRemoteIdentity(cfg.Agent.CredentialFile, identity)
				client, _ = remoteHTTPClient(identity)
			} else {
				log.Warn().Err(rotateErr).Msg("agent certificate rotation failed")
			}
		}
		if identity.PendingResult != nil {
			if err := postCommandResult(ctx, client, cfg.Agent.ServerURL, identity, identity.PendingResult.CommandID, identity.PendingResult.Response); err != nil {
				log.Warn().Err(err).Msg("agent result delivery failed")
				if !waitContext(ctx, backoff) {
					return nil
				}
				continue
			}
			identity.PendingResult = nil
			_ = saveRemoteIdentity(cfg.Agent.CredentialFile, identity)
		}
		identity.Sequence++
		state := s.remoteState(ctx, identity.Sequence, identity.PolicyRevision)
		body, _ := json.Marshal(state)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Agent.ServerURL+"/api/v1/agent/poll", bytes.NewReader(body))
		authorizeAgent(req, identity)
		resp, err := client.Do(req)
		if err != nil {
			log.Warn().Err(err).Msg("agent sync disconnected")
			if !waitContext(ctx, backoff) {
				return nil
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		if resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			backoff = time.Second
			_ = saveRemoteIdentity(cfg.Agent.CredentialFile, identity)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			msg := limitedBody(resp.Body)
			resp.Body.Close()
			log.Warn().Int("status", resp.StatusCode).Str("response", msg).Msg("agent poll rejected")
			if !waitContext(ctx, backoff) {
				return nil
			}
			continue
		}
		var cmd database.AgentCommand
		err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&cmd)
		resp.Body.Close()
		if err != nil {
			continue
		}
		result := s.dispatch(ctx, Request{Op: cmd.Op, Params: cmd.Params})
		if result.OK && (cmd.Op == "ApplyTransaction" || cmd.Op == "RestoreSnapshot" || cmd.Op == "Reload" || cmd.Op == "Restart") {
			identity.PolicyRevision++
		}
		identity.PendingResult = &pendingResult{CommandID: cmd.CommandID, Response: result}
		_ = saveRemoteIdentity(cfg.Agent.CredentialFile, identity)
		if err := postCommandResult(ctx, client, cfg.Agent.ServerURL, identity, cmd.CommandID, result); err == nil {
			identity.PendingResult = nil
			_ = saveRemoteIdentity(cfg.Agent.CredentialFile, identity)
		}
		backoff = time.Second
	}
}

func (s *Server) remoteState(ctx context.Context, seq, revision int64) database.AgentState {
	status := s.dispatch(ctx, Request{Op: "GetFirewallStatus"})
	rules := s.dispatch(ctx, Request{Op: "ListRules"})
	caps := s.dispatch(ctx, Request{Op: "GetCapabilities"})
	h := sha256.New()
	h.Write(status.Payload)
	h.Write(rules.Payload)
	health := "HEALTHY"
	if !status.OK || !rules.OK {
		health = "DEGRADED"
	}
	var capData struct {
		Backend   string `json:"backend"`
		Ownership struct {
			Owner string `json:"owner"`
		} `json:"ownership"`
	}
	_ = json.Unmarshal(caps.Payload, &capData)
	return database.AgentState{Sequence: seq, StateHash: hex.EncodeToString(h.Sum(nil)), PolicyRevision: revision, HealthState: health, FirewallBackend: capData.Backend, FirewallOwner: capData.Ownership.Owner, Status: status.Payload, Rules: rules.Payload, Capabilities: caps.Payload}
}

func enrollRemote(ctx context.Context, cfg *config.Config, id string) (remoteIdentity, error) {
	if cfg.Agent.EnrollmentToken == "" {
		return remoteIdentity{}, fmt.Errorf("remote agent is not enrolled; set FYRWALL_AGENT_ENROLLMENT_TOKEN")
	}
	host, _ := os.Hostname()
	p := system.DetectPlatform()
	display := cfg.Agent.DisplayName
	if display == "" {
		display = host
	}
	payload := enrollRequestWire{Token: cfg.Agent.EnrollmentToken, Agent: database.AgentRecord{AgentID: id, DisplayName: display, Hostname: host, Architecture: p.Arch, Distribution: p.DistroName, KernelVersion: p.KernelVersion, AgentVersion: version.Version, Labels: cfg.Agent.Labels, Groups: cfg.Agent.Groups, Site: cfg.Agent.Site}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Agent.ServerURL+"/api/v1/agent/enroll", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return remoteIdentity{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return remoteIdentity{}, fmt.Errorf("enrollment failed: HTTP %d: %s", resp.StatusCode, limitedBody(resp.Body))
	}
	var out remoteIdentity
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

type enrollRequestWire struct {
	Token string               `json:"token"`
	Agent database.AgentRecord `json:"agent"`
}

func remoteHTTPClient(identity remoteIdentity) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if identity.Certificate != "" && identity.PrivateKey != "" {
		cert, err := tls.X509KeyPair([]byte(identity.Certificate), []byte(identity.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("agent client certificate: %w", err)
		}
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}
	}
	return &http.Client{Timeout: 35 * time.Second, Transport: transport}, nil
}

func certificateNeedsRotation(certPEM string, within time.Duration) bool {
	if certPEM == "" {
		return false
	}
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	return err != nil || time.Until(cert.NotAfter) < within
}

func rotateRemoteCertificate(ctx context.Context, c *http.Client, base string, id remoteIdentity) (remoteIdentity, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/agent/rotate-certificate", bytes.NewReader([]byte(`{}`)))
	authorizeAgent(req, id)
	resp, err := c.Do(req)
	if err != nil {
		return id, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return id, fmt.Errorf("certificate rotation rejected: HTTP %d", resp.StatusCode)
	}
	var out struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"private_key"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return id, err
	}
	id.Certificate, id.PrivateKey = out.Certificate, out.PrivateKey
	return id, nil
}

func postCommandResult(ctx context.Context, c *http.Client, base string, id remoteIdentity, commandID string, result Response) error {
	body, _ := json.Marshal(map[string]any{"command_id": commandID, "result": result.Payload, "error": result.Error})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/agent/result", bytes.NewReader(body))
	authorizeAgent(req, id)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("result rejected: HTTP %d", resp.StatusCode)
	}
	return nil
}
func authorizeAgent(req *http.Request, id remoteIdentity) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FYRwall-Agent-ID", id.AgentID)
	req.Header.Set("Authorization", "Bearer "+id.Credential)
}
func loadAgentID(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
		return strings.TrimSpace(string(b)), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", err
	}
	id := uuid.NewString()
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	return id, nil
}
func loadRemoteIdentity(path string) (remoteIdentity, error) {
	var v remoteIdentity
	b, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
func saveRemoteIdentity(path string, v remoteIdentity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func limitedBody(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 4096))
	return strings.TrimSpace(string(b))
}
func waitContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
