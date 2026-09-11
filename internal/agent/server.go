// Package agent implements the unprivileged local firewall agent: a
// Unix-socket server exposing only strongly typed operations (spec
// sections 5.2, 42). No generic exec method exists.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

// SocketPath is the default control socket.
const SocketPath = "/run/fyrwall/agent.sock"

// Request is a typed agent request; Op is one of the allowlisted
// operations below. Never a raw command string.
type Request struct {
	Op     string          `json:"op"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the typed reply.
type Response struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Allowlisted operations (spec section 42). Anything else is rejected.
var allowedOps = map[string]bool{
	"GetCapabilities":      true,
	"GetFirewallStatus":    true,
	"ListRules":            true,
	"ValidateTransaction":  true,
	"CreateSnapshot":       true,
	"ApplyTransaction":     true,
	"VerifyState":          true,
	"RestoreSnapshot":      true,
	"RunAllowedDiagnostic": true,
}

// Server hosts the Unix socket and dispatches typed requests to the
// firewall manager.
type Server struct {
	mgr      *firewall.Manager
	log      zerolog.Logger
	ln       net.Listener
	mu       sync.Mutex
	stopping bool
}

// NewServer builds the agent server.
func NewServer(mgr *firewall.Manager, log zerolog.Logger) *Server {
	return &Server{mgr: mgr, log: log}
}

// Listen binds the socket with strict permissions: parent dir 0750 and
// socket 0660 (spec sections 5.2, 8.5).
func (s *Server) Listen(path string) error {
	if path == "" {
		path = SocketPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("socket dir: %w", err)
	}
	// Remove stale socket from a crashed previous run.
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("listen %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o660); err != nil {
		ln.Close()
		return err
	}
	s.ln = ln
	return nil
}

// Serve accepts connections until ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		s.stopping = true
		s.mu.Unlock()
		s.ln.Close()
	}()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			stopping := s.stopping
			s.mu.Unlock()
			if stopping {
				return nil
			}
			return err
		}
		go s.handleConn(ctx, conn)
	}
}

// handleConn serves one connection with per-request deadlines.
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	// Bound request size (spec section 82).
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var req Request
	if err := dec.Decode(&req); err != nil {
		writeResp(conn, Response{OK: false, Error: "malformed request"})
		return
	}
	if !allowedOps[req.Op] {
		s.log.Warn().Str("op", req.Op).Msg("agent rejected non-allowlisted op")
		writeResp(conn, Response{OK: false, Error: "operation not allowed"})
		return
	}
	ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	resp := s.dispatch(ctx2, req)
	writeResp(conn, resp)
}

// dispatch routes an allowlisted op to the firewall manager.
func (s *Server) dispatch(ctx context.Context, req Request) Response {
	switch req.Op {
	case "GetCapabilities":
		return okResp(map[string]any{
			"backend":    s.mgr.Ownership().Owner,
			"operations": allowedOpNames(),
			"protocol":   1,
		})
	case "GetFirewallStatus":
		st, err := s.mgr.Status(ctx)
		if err != nil {
			return errResp(err)
		}
		return okResp(st)
	case "ListRules":
		rules, err := s.mgr.ListRules(ctx)
		if err != nil {
			return errResp(err)
		}
		return okResp(map[string]any{"rules": rules})
	case "ValidateTransaction":
		var tx firewall.Transaction
		if err := json.Unmarshal(req.Params, &tx); err != nil {
			return errResp(err)
		}
		return okResp(s.mgr.BackendValidate(ctx, tx))
	case "CreateSnapshot":
		var p struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return errResp(err)
		}
		snap, err := s.mgr.CreateSnapshot(ctx, "agent", p.Reason)
		if err != nil {
			return errResp(err)
		}
		return okResp(snap)
	case "ApplyTransaction":
		var tx firewall.Transaction
		if err := json.Unmarshal(req.Params, &tx); err != nil {
			return errResp(err)
		}
		res, err := s.mgr.ApplyTransaction(ctx, "agent", tx)
		if err != nil {
			return errResp(err)
		}
		return okResp(res)
	case "RunAllowedDiagnostic":
		// Read-only diagnostics only.
		return okResp(map[string]any{"supported": true})
	default:
		return Response{OK: false, Error: "operation not allowed"}
	}
}

func allowedOpNames() []string {
	out := make([]string, 0, len(allowedOps))
	for k := range allowedOps {
		out = append(out, k)
	}
	return out
}

func okResp(v any) Response {
	b, err := json.Marshal(v)
	if err != nil {
		return Response{OK: false, Error: "encode failure"}
	}
	return Response{OK: true, Payload: b}
}

func errResp(err error) Response {
	// Sanitized: error text only, never raw stderr (spec section 43).
	return Response{OK: false, Error: err.Error()}
}

func writeResp(conn net.Conn, r Response) {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	json.NewEncoder(conn).Encode(r)
}
