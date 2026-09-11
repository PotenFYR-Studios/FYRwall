package api

import (
	"net/http"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
)

// handleLogin authenticates a local user with rate limiting, audit and
// session fixation defense (fresh session ID on login, spec section 77).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	ip := auth.NormalizeIP(r.RemoteAddr)
	if !s.limiter.Allow(ip) {
		s.log.Warn().Str("ip", ip).Str("event_code", "AUTH_RATE_LIMITED").Msg("login rate limited")
		writeErr(w, r, http.StatusTooManyRequests, "AUTH_RATE_LIMITED",
			"too many login attempts; wait before retrying", nil)
		return
	}

	u, err := s.users.Get(body.Username)
	if err != nil {
		s.auditFailure(body.Username, ip, "user lookup failed")
		writeErr(w, r, http.StatusUnauthorized, "AUTH_FORBIDDEN", "invalid credentials", nil)
		return
	}
	ok, _, err := auth.VerifyPassword(body.Password, u.PasswordHash)
	if err != nil || !ok {
		s.auditFailure(u.Username, ip, "bad password")
		writeErr(w, r, http.StatusUnauthorized, "AUTH_FORBIDDEN", "invalid credentials", nil)
		return
	}
	if !u.Enabled {
		s.auditFailure(u.Username, ip, "disabled user")
		writeErr(w, r, http.StatusUnauthorized, "AUTH_FORBIDDEN", "invalid credentials", nil)
		return
	}

	// Fresh session token on login (session fixation defense).
	s.sessions.Manager().Destroy(r.Context())
	if err := s.sessions.Manager().RenewToken(r.Context()); err != nil {
		writeErr(w, r, http.StatusInternalServerError, "INTERNAL", "session error", nil)
		return
	}
	s.sessions.Manager().Put(r.Context(), "username", u.Username)
	s.sessions.Manager().Put(r.Context(), "role", string(u.Role))
	s.sessions.Manager().Put(r.Context(), "login_at", time.Now().UTC().Format(time.RFC3339Nano))
	s.sessions.EnsureCSRF(r)
	s.users.TouchLogin(u.ID)
	s.audit.Insert(database.AuditEntry{
		Actor: u.Username, SourceIP: ip, Action: "auth.login", Success: true,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"username": u.Username, "role": u.Role,
		"must_change_password": u.MustChangePass,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if user := s.currentUser(r); user != "" {
		s.audit.Insert(database.AuditEntry{Actor: user, Action: "auth.logout", Success: true})
	}
	s.sessions.Manager().Destroy(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == "" {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	role := s.sessions.Manager().GetString(r.Context(), "role")
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true, "username": user, "role": role,
		"csrf_token": s.sessions.EnsureCSRF(r),
	})
}

// requireAuth rejects anonymous requests to management APIs (spec section
// 13: authentication is mandatory).
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.currentUser(r)
		if user == "" {
			writeErr(w, r, http.StatusUnauthorized, "AUTH_FORBIDDEN", "authentication required", nil)
			return
		}
		// Revocation check: sessions invalidated by admin action.
		loginStr := s.sessions.Manager().GetString(r.Context(), "login_at")
		if loginStr != "" {
			if loginAt, err := time.Parse(time.RFC3339Nano, loginStr); err == nil {
				if auth.SessionRevokedAfter(user, loginAt) {
					s.sessions.Manager().Destroy(r.Context())
					writeErr(w, r, http.StatusUnauthorized, "AUTH_FORBIDDEN", "session revoked", nil)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireCSRF validates the CSRF token on state-changing methods.
func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.sessions.CheckCSRF(r) {
			writeErr(w, r, http.StatusForbidden, "CSRF_FAILED", "missing or invalid CSRF token", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requirePerm enforces RBAC in the backend, not only the UI (spec rule 10
// of section 52).
func (s *Server) requirePerm(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := auth.Role(s.sessions.Manager().GetString(r.Context(), "role"))
			if !auth.HasPermission(role, perm) {
				user := s.currentUser(r)
				s.log.Warn().Str("user", user).Str("perm", perm).Msg("authorization denied")
				writeErr(w, r, http.StatusForbidden, "AUTH_FORBIDDEN",
					"insufficient permissions for "+perm, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) currentUser(r *http.Request) string {
	return s.sessions.Manager().GetString(r.Context(), "username")
}

func (s *Server) auditFailure(user, ip, reason string) {
	s.audit.Insert(database.AuditEntry{
		Actor: user, SourceIP: ip, Action: "auth.login", Success: false, Detail: reason,
	})
	s.log.Info().Str("user", user).Str("ip", ip).Str("event_code", "AUTH_LOGIN_FAILED").Msg(reason)
}
