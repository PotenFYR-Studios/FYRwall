package api

import (
	"net/http"
	"strings"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
)

// One-time super-admin setup: on a fresh install (Docker or host) no
// credentials exist, so the first visitor through the web GUI claims the
// super admin account and chooses its password. The CLI never handles
// passwords.

// setupNeeded reports whether the one-time setup has not been completed.
func (s *Server) setupNeeded() (bool, error) {
	n, err := s.users.CountAdmins()
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// handleSetupStatus is public: the GUI needs it to route to the wizard or
// the login page. It exposes no credentials, only a boolean.
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	needed, err := s.setupNeeded()
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "INTERNAL", "setup status unavailable", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"setup_needed": needed})
}

// handleSetupSuperAdmin claims the super admin account on first run.
// Rate limited by the same limiter as login: it creates credentials.
func (s *Server) handleSetupSuperAdmin(w http.ResponseWriter, r *http.Request) {
	needed, err := s.setupNeeded()
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "INTERNAL", "setup status unavailable", nil)
		return
	}
	if !needed {
		writeErr(w, r, http.StatusConflict, "SETUP_DONE",
			"super admin already set up; sign in instead", nil)
		return
	}
	ip := auth.NormalizeIP(r.RemoteAddr)
	if !s.limiter.Allow(ip) {
		s.log.Warn().Str("ip", ip).Str("event_code", "AUTH_RATE_LIMITED").Msg("setup rate limited")
		writeErr(w, r, http.StatusTooManyRequests, "AUTH_RATE_LIMITED",
			"too many attempts; wait before retrying", nil)
		return
	}

	var body struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if body.Password != body.ConfirmPassword {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", "passwords do not match", nil)
		return
	}
	if fails := auth.ValidatePasswordDefault(body.Password); len(fails) > 0 {
		writeErr(w, r, http.StatusBadRequest, "PASSWORD_POLICY",
			"password policy: "+strings.Join(fails, "; "), nil)
		return
	}

	hash, err := auth.HashPassword(body.Password, auth.DefaultArgonParams())
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "INTERNAL", "hash error", nil)
		return
	}
	u, err := s.users.Create(body.Username, hash, auth.RoleSuperAdmin, false)
	if err != nil {
		s.log.Warn().Err(err).Str("event_code", "SETUP_FAILED").Msg("super admin setup failed")
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID",
			"could not create super admin (username taken or invalid)", nil)
		return
	}

	// Audit without a session actor.
	s.audit.Insert(database.AuditEntry{
		Actor: u.Username, SourceIP: ip, Action: "auth.setup_super_admin", Success: true,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"username": u.Username, "role": u.Role})
}
