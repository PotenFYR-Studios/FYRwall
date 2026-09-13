package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
)

type enrollRequest struct {
	Token string               `json:"token"`
	Agent database.AgentRecord `json:"agent"`
}

func (s *Server) handleAgentEnroll(w http.ResponseWriter, r *http.Request) {
	var body enrollRequest
	if err := decodeStrict(r, &body); err != nil || body.Token == "" || body.Agent.AgentID == "" || body.Agent.Hostname == "" {
		writeErr(w, r, http.StatusBadRequest, "AGENT_ENROLL_INVALID", "token, agent_id and hostname are required", nil)
		return
	}
	credential := auth.RandomTokenHex(32)
	if err := s.fleet.Enroll(body.Token, body.Agent, credential); err != nil {
		writeErr(w, r, http.StatusUnauthorized, "AGENT_ENROLL_DENIED", err.Error(), nil)
		return
	}
	var certPEM, keyPEM string
	if s.agentPKI != nil {
		var fingerprint string
		var expiry time.Time
		var issueErr error
		certPEM, keyPEM, fingerprint, expiry, issueErr = s.agentPKI.Issue(body.Agent.AgentID, 90*24*time.Hour)
		if issueErr != nil {
			writeErr(w, r, http.StatusInternalServerError, "AGENT_CERT_FAILED", issueErr.Error(), nil)
			return
		}
		if issueErr = s.fleet.SetAgentCertificate(body.Agent.AgentID, fingerprint, expiry); issueErr != nil {
			writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", issueErr.Error(), nil)
			return
		}
	}
	s.audit.Insert(database.AuditEntry{Actor: "agent:" + body.Agent.AgentID, Action: "agent.enroll", Target: body.Agent.AgentID, Success: true})
	writeJSON(w, http.StatusCreated, map[string]any{"agent_id": body.Agent.AgentID, "credential": credential, "certificate": certPEM, "private_key": keyPEM, "poll_interval_seconds": 1})
}

func agentCredentials(r *http.Request) (string, string) {
	id := r.Header.Get("X-FYRwall-Agent-ID")
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		return id, ""
	}
	return id, strings.TrimPrefix(authz, "Bearer ")
}

func (s *Server) authenticateAgent(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, credential := agentCredentials(r)
	if id == "" || credential == "" || !s.fleet.Authenticate(id, credential) {
		writeErr(w, r, http.StatusUnauthorized, "AGENT_AUTH_FAILED", "invalid agent credentials", nil)
		return "", false
	}
	if s.cfg.TLS.Enabled {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			writeErr(w, r, http.StatusUnauthorized, "AGENT_CERT_REQUIRED", "valid agent client certificate required", nil)
			return "", false
		}
		cert := r.TLS.PeerCertificates[0]
		sum := sha256.Sum256(cert.Raw)
		if cert.Subject.CommonName != id || !s.fleet.CertificateMatches(id, hex.EncodeToString(sum[:])) {
			writeErr(w, r, http.StatusUnauthorized, "AGENT_CERT_REVOKED", "agent certificate rejected", nil)
			return "", false
		}
	}
	return id, true
}

func (s *Server) handleAgentPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := s.authenticateAgent(w, r)
	if !ok {
		return
	}
	var state database.AgentState
	if err := decodeStrict(r, &state); err != nil {
		writeErr(w, r, http.StatusBadRequest, "AGENT_STATE_INVALID", err.Error(), nil)
		return
	}
	drift, err := s.fleet.UpdateState(id, state)
	if err != nil {
		writeErr(w, r, http.StatusConflict, "AGENT_STATE_STALE", err.Error(), nil)
		return
	}
	if drift {
		_ = s.notifs.Upsert(database.Notification{Code: "AGENT_FIREWALL_DRIFT", Severity: "warning", Category: "fleet", Component: "agent:" + id, Summary: "Firewall state changed outside a FYRwall policy operation", Remediation: "Review target rules and either accept the new baseline or reapply the intended policy.", Fingerprint: "AGENT_FIREWALL_DRIFT|" + id})
	}
	deadline := time.NewTimer(20 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		cmd, err := s.fleet.NextCommand(id)
		if err != nil {
			writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
			return
		}
		if cmd != nil {
			writeJSON(w, http.StatusOK, cmd)
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			w.WriteHeader(http.StatusNoContent)
			return
		case <-tick.C:
		}
	}
}

func (s *Server) handleAgentResult(w http.ResponseWriter, r *http.Request) {
	id, ok := s.authenticateAgent(w, r)
	if !ok {
		return
	}
	var body struct {
		CommandID string          `json:"command_id"`
		Result    json.RawMessage `json:"result,omitempty"`
		Error     string          `json:"error,omitempty"`
	}
	if err := decodeStrict(r, &body); err != nil || body.CommandID == "" {
		writeErr(w, r, http.StatusBadRequest, "AGENT_RESULT_INVALID", "command_id required", nil)
		return
	}
	if err := s.fleet.CompleteCommand(id, body.CommandID, body.Result, body.Error); err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAgentCertificateRotate(w http.ResponseWriter, r *http.Request) {
	id, ok := s.authenticateAgent(w, r)
	if !ok {
		return
	}
	if s.agentPKI == nil {
		writeErr(w, r, http.StatusServiceUnavailable, "AGENT_CERT_UNAVAILABLE", "agent certificate authority unavailable", nil)
		return
	}
	cert, key, fingerprint, expiry, err := s.agentPKI.Issue(id, 90*24*time.Hour)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "AGENT_CERT_FAILED", err.Error(), nil)
		return
	}
	if err = s.fleet.RotateAgentCertificate(id, fingerprint, expiry); err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"certificate": cert, "private_key": key, "expires_at": expiry})
}

func (s *Server) handleEnrollmentTokenCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentID    string `json:"agent_id,omitempty"`
		TTLMinutes int    `json:"ttl_minutes,omitempty"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if body.TTLMinutes == 0 {
		body.TTLMinutes = 15
	}
	if body.TTLMinutes < 1 || body.TTLMinutes > 1440 {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", "ttl_minutes must be 1..1440", nil)
		return
	}
	token, expires, err := s.fleet.CreateEnrollmentToken(body.AgentID, time.Duration(body.TTLMinutes)*time.Minute)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{Actor: s.currentUser(r), Action: "agent.enrollment_token.create", Target: body.AgentID, Success: true})
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "expires_at": expires})
}

func (s *Server) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	agents, err := s.fleet.ListAgents()
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

func (s *Server) handleRemoteFirewallStatus(w http.ResponseWriter, r *http.Request) {
	a, err := s.fleet.GetAgent(chi.URLParam(r, "agentID"))
	if err != nil {
		writeErr(w, r, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found", nil)
		return
	}
	if len(a.Status) == 0 {
		writeErr(w, r, http.StatusServiceUnavailable, "AGENT_STATE_UNAVAILABLE", "agent has not synchronized status", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": a.Status, "owner": a.FirewallOwner, "state_hash": a.StateHash, "policy_revision": a.PolicyRevision, "last_seen_at": a.LastSeenAt})
}

func (s *Server) handleRemoteFirewallRules(w http.ResponseWriter, r *http.Request) {
	a, err := s.fleet.GetAgent(chi.URLParam(r, "agentID"))
	if err != nil {
		writeErr(w, r, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found", nil)
		return
	}
	if len(a.Rules) == 0 {
		writeErr(w, r, http.StatusServiceUnavailable, "AGENT_STATE_UNAVAILABLE", "agent has not synchronized rules", nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.Rules)
}

func (s *Server) handleRemoteFirewallApply(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || !json.Valid(b) {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", "valid transaction JSON required", nil)
		return
	}
	cmd, err := s.fleet.QueueCommand(chi.URLParam(r, "agentID"), "ApplyTransaction", json.RawMessage(b))
	if err != nil {
		writeErr(w, r, http.StatusConflict, "AGENT_COMMAND_FAILED", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{Actor: s.currentUser(r), Action: "agent.firewall.apply.queue", Target: chi.URLParam(r, "agentID"), Success: true, Detail: cmd.CommandID})
	writeJSON(w, http.StatusAccepted, cmd)
}

var remoteOps = map[string]bool{"GetCapabilities": true, "GetFirewallStatus": true, "ListRules": true, "ValidateTransaction": true, "CreateSnapshot": true, "ApplyTransaction": true, "VerifyState": true, "RestoreSnapshot": true, "RestoreLast": true, "RunAllowedDiagnostic": true, "Reload": true, "Restart": true}

func (s *Server) handleAgentCommandCreate(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	var body struct {
		Op     string          `json:"op"`
		Params json.RawMessage `json:"params,omitempty"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if !remoteOps[body.Op] {
		writeErr(w, r, http.StatusBadRequest, "AGENT_OP_DENIED", "operation not allowlisted", nil)
		return
	}
	cmd, err := s.fleet.QueueCommand(agentID, body.Op, body.Params)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{Actor: s.currentUser(r), Action: "agent.command.queue", Target: agentID, Success: true, Detail: body.Op})
	writeJSON(w, http.StatusAccepted, cmd)
}

func (s *Server) handleAgentCommandGet(w http.ResponseWriter, r *http.Request) {
	cmd, err := s.fleet.GetCommand(chi.URLParam(r, "commandID"))
	if err != nil {
		writeErr(w, r, http.StatusNotFound, "NOT_FOUND", "command not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, cmd)
}

func (s *Server) handleRolloutCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Op          string          `json:"op"`
		Params      json.RawMessage `json:"params,omitempty"`
		BatchSize   int             `json:"batch_size"`
		MaxFailures int             `json:"max_failures"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if !remoteOps[body.Op] {
		writeErr(w, r, http.StatusBadRequest, "AGENT_OP_DENIED", "operation not allowlisted", nil)
		return
	}
	rollout, err := s.fleet.CreateRollout(chi.URLParam(r, "group"), body.Op, body.Params, body.BatchSize, body.MaxFailures)
	if err != nil {
		writeErr(w, r, http.StatusConflict, "ROLLOUT_CREATE_FAILED", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{Actor: s.currentUser(r), Action: "fleet.rollout.create", Target: rollout.TargetGroup, Success: true, Detail: rollout.RolloutID})
	writeJSON(w, http.StatusAccepted, rollout)
}

func (s *Server) handleRolloutGet(w http.ResponseWriter, r *http.Request) {
	rollout, err := s.fleet.GetRollout(chi.URLParam(r, "rolloutID"))
	if err != nil {
		writeErr(w, r, http.StatusNotFound, "ROLLOUT_NOT_FOUND", "rollout not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, rollout)
}

func (s *Server) handleAgentRevoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "agentID")
	if err := s.fleet.RevokeAgent(id); err != nil {
		writeErr(w, r, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found", nil)
		return
	}
	s.audit.Insert(database.AuditEntry{Actor: s.currentUser(r), Action: "agent.revoke", Target: id, Success: true})
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}
