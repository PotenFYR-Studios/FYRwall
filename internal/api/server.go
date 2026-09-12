// Package api assembles the HTTP surface: versioned routes, security
// middleware, JSON envelopes and machine-readable errors (spec sections
// 27, 35, 45). No firewall logic lives in handlers; everything routes
// through internal/firewall.Manager.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/diagnostics"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/health"
	"github.com/PotenFYR-Studios/FYRwall/internal/secureconfig"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
	"github.com/PotenFYR-Studios/FYRwall/webembed"
	"gopkg.in/yaml.v3"
)

// Server bundles everything the handlers need.
type Server struct {
	cfg      *config.Config
	log      zerolog.Logger
	db       *database.DB
	users    *database.UserRepo
	audit    *database.AuditRepo
	notifs   *database.NotificationRepo
	health   *health.Tracker
	firewall *firewall.Manager
	diag     *diagnostics.Registry
	sessions *auth.SessionManager
	limiter  *auth.LoginRateLimiter
}

// New builds the API server.
func New(cfg *config.Config, log zerolog.Logger, db *database.DB, fw *firewall.Manager, ht *health.Tracker, diag *diagnostics.Registry) *Server {
	return &Server{
		cfg:      cfg,
		log:      log,
		db:       db,
		users:    database.NewUserRepo(db),
		audit:    database.NewAuditRepo(db),
		notifs:   database.NewNotificationRepo(db),
		health:   ht,
		firewall: fw,
		diag:     diag,
		sessions: auth.NewSessionManager(
			time.Duration(cfg.Security.SessionIdleTimeoutMinutes)*time.Minute,
			cfg.TLS.Enabled,
		),
		limiter: auth.NewLoginRateLimiter(cfg.Security.LoginRateLimitPerMinute),
	}
}

// Router assembles the middleware chain and all routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(chiMW.RequestID)
	r.Use(chiMW.RealIP)
	r.Use(securityHeaders(s.cfg))
	r.Use(chiMW.Timeout(30 * time.Second))
	r.Use(s.sessions.Manager().LoadAndSave)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // dev only; prod is same-origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", auth.CSRFHeader, "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(bodyLimit(1 << 20)) // 1 MiB cap, spec section 77
	if s.cfg.Server.Domain != "" {
		r.Use(hostAllowlist(s.cfg.Server.Domain))
	}

	// Embedded SPA (spec sections 3.2, 72): same-origin UI, no Node on the
	// production host. Mounted first so / serves index.html; /api routes
	// below take precedence by explicit registration.
	if ui, err := webembed.Handler(); err == nil {
		r.Handle("/*", ui)
	} else {
		// Fail loudly instead of serving a blank page at "/".
		s.log.Error().Str("event_code", "WEB_UI_MISSING").Err(err).
			Msg("embedded web GUI unavailable; running in API-only mode")
	}

	// Public: health (unauthenticated, minimal data) + version.
	r.Get("/api/v1/version", handleVersion)
	r.Get("/api/v1/system/health", s.handleHealth)
	r.Post("/api/v1/auth/login", s.handleLogin)
	r.Post("/api/v1/auth/logout", s.handleLogout)
	r.Get("/api/v1/auth/me", s.handleMe)

	// Authenticated API.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Use(s.requireCSRF)

		r.With(s.requirePerm(auth.PermFirewallRead)).Get("/firewall/status", s.handleFirewallStatus)
		r.With(s.requirePerm(auth.PermFirewallRead)).Get("/firewall/rules", s.handleFirewallRules)
		r.With(s.requirePerm(auth.PermFirewallApply)).Post("/firewall/rules/validate", s.handleFirewallValidate)
		r.With(s.requirePerm(auth.PermFirewallApply)).Post("/firewall/transactions", s.handleFirewallApply)
		r.With(s.requirePerm(auth.PermFirewallRestore)).Post("/firewall/snapshots", s.handleSnapshotCreate)

		r.With(s.requirePerm(auth.PermSettingsRead)).Get("/settings", s.handleSettingsGet)
		r.With(s.requirePerm(auth.PermSettingsWrite)).Put("/settings", s.handleSettingsPut)

		r.With(s.requirePerm(auth.PermUsersManage)).Get("/users", s.handleUsersList)
		r.With(s.requirePerm(auth.PermUsersManage)).Post("/users", s.handleUsersCreate)

		r.With(s.requirePerm(auth.PermDiagnosticsRun)).Get("/system/diagnostics", s.handleDiagnostics)
		r.With(s.requirePerm(auth.PermLogsRead)).Get("/audit", s.handleAuditList)
		r.Get("/notifications", s.handleNotificationsList)
	})

	return r
}

// securityHeaders applies the hardening headers from spec section 35.
func securityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	csp := "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
		"connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if cfg.TLS.Enabled {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hostAllowlist enforces the configured domain binding: requests with a
// foreign Host header (direct IP scans, host-header confusion) get 404
// with no server identity leaked.
func hostAllowlist(domain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			if i := strings.LastIndex(host, ":"); i > 0 && !strings.HasSuffix(host, "]") {
				// strip port; IPv6 literals keep brackets
				host = host[:i]
			}
			if !strings.EqualFold(host, domain) {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bodyLimit caps request body size (oversized request defense).
func bodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---- JSON envelope helpers (spec section 27) ----

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	ReqID   string         `json:"request_id,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, r *http.Request, status int, code, msg string, details map[string]any) {
	reqID, _ := r.Context().Value(chiMW.RequestIDKey).(string)
	writeJSON(w, status, map[string]any{"error": apiError{
		Code: code, Message: msg, Details: details, ReqID: reqID,
	}})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, version.Info())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	overall, components := s.health.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"state":      overall,
		"components": components,
		"uptime_s":   int(s.health.Uptime().Seconds()),
	})
}

func (s *Server) handleFirewallStatus(w http.ResponseWriter, r *http.Request) {
	st, err := s.firewall.Status(r.Context())
	if err != nil {
		if errors.Is(err, firewall.ErrBackendUnavailable) {
			writeErr(w, r, http.StatusServiceUnavailable, "FW_BACKEND_NOT_FOUND", err.Error(), nil)
			return
		}
		writeErr(w, r, http.StatusInternalServerError, "FW_COMMAND_FAILED", "failed to read firewall status", nil)
		return
	}
	st.RuleCount = 0
	rules, _ := s.firewall.ListRules(r.Context())
	st.RuleCount = len(rules)
	own := s.firewall.Ownership()
	writeJSON(w, http.StatusOK, map[string]any{
		"status": st, "ownership": own, "writes_allowed": own.WritesAllowed,
	})
}

func (s *Server) handleFirewallRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.firewall.ListRules(r.Context())
	if err != nil {
		writeErr(w, r, http.StatusServiceUnavailable, "FW_BACKEND_NOT_FOUND", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "count": len(rules)})
}

func (s *Server) handleFirewallValidate(w http.ResponseWriter, r *http.Request) {
	var tx firewall.Transaction
	if err := decodeStrict(r, &tx); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	vr := s.firewall.BackendValidate(r.Context(), tx)
	writeJSON(w, http.StatusOK, vr)
}

func (s *Server) handleFirewallApply(w http.ResponseWriter, r *http.Request) {
	var tx firewall.Transaction
	if err := decodeStrict(r, &tx); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	actor := s.currentUser(r)
	res, err := s.firewall.ApplyTransaction(r.Context(), actor, tx)
	if err != nil {
		var vf *firewall.ValidationFailure
		if errors.As(err, &vf) {
			writeErr(w, r, http.StatusUnprocessableEntity, "FW_VALIDATION_FAILED",
				"validation failed", map[string]any{"errors": vf.Errors, "conflicts": vf.Conflicts})
			return
		}
		if errors.Is(err, firewall.ErrWritesBlocked) {
			writeErr(w, r, http.StatusConflict, "FW_BACKEND_CONFLICT", err.Error(), nil)
			return
		}
		writeErr(w, r, http.StatusInternalServerError, "FW_APPLY_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSnapshotCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	snap, err := s.firewall.CreateSnapshot(r.Context(), s.currentUser(r), body.Reason)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "FW_SNAPSHOT_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"bind": s.cfg.Server.Bind, "port": s.cfg.Server.Port,
		"tls": s.cfg.TLS.Enabled, "backend": s.cfg.Firewall.Backend,
		"autostart": s.cfg.Service.Autostart,
	})
}

// handleSettingsPut is the ONLY supported config write path (web GUI,
// admin permission + CSRF). The new config is validated, then persisted
// encrypted via secureconfig; plaintext never touches disk.
func (s *Server) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Bind    string `json:"bind,omitempty"`
		Port    *int   `json:"port,omitempty"`
		Domain  string `json:"domain,omitempty"`
		Updates *bool  `json:"updates_check_enabled,omitempty"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if body.Bind != "" {
		s.cfg.Server.Bind = body.Bind
	}
	if body.Port != nil {
		s.cfg.Server.Port = *body.Port
	}
	if body.Domain != "" {
		s.cfg.Server.Domain = body.Domain
	}
	if body.Updates != nil {
		s.cfg.Updates.CheckEnabled = *body.Updates
	}
	// Validate BEFORE persisting anything.
	if err := s.cfg.Validate(); err != nil {
		writeErr(w, r, http.StatusUnprocessableEntity, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	sc, err := secureconfig.New(config.ConfigPath())
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "CONFIG_STORE", err.Error(), nil)
		return
	}
	b, err := yaml.Marshal(s.cfg)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "CONFIG_STORE", err.Error(), nil)
		return
	}
	if err := sc.Save(b); err != nil {
		writeErr(w, r, http.StatusInternalServerError, "CONFIG_STORE", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{
		Actor: s.currentUser(r), Action: "settings.write", Success: true,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := s.users.List()
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	type safeUser struct {
		ID       string     `json:"id"`
		Username string     `json:"username"`
		Role     string     `json:"role"`
		Enabled  bool       `json:"enabled"`
		LastSeen *time.Time `json:"last_login_at"`
	}
	out := make([]safeUser, 0, len(users))
	for _, u := range users {
		out = append(out, safeUser{ID: u.ID, Username: u.Username, Role: string(u.Role), Enabled: u.Enabled, LastSeen: u.LastLoginAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *Server) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID", err.Error(), nil)
		return
	}
	if len(body.Username) < 3 {
		writeErr(w, r, http.StatusBadRequest, "CONFIG_INVALID",
			"username min 3 chars", nil)
		return
	}
	if fails := auth.ValidatePasswordDefault(body.Password); len(fails) > 0 {
		writeErr(w, r, http.StatusBadRequest, "PASSWORD_POLICY",
			"password policy: "+strings.Join(fails, "; "), nil)
		return
	}
	hash, err := auth.HashPassword(body.Password, auth.DefaultArgonParams())
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "INTERNAL", "hash failure", nil)
		return
	}
	u, err := s.users.Create(body.Username, hash, auth.Role(body.Role), false)
	if err != nil {
		writeErr(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
		return
	}
	s.audit.Insert(database.AuditEntry{
		Actor: s.currentUser(r), Action: "user.create", Target: u.ID, Success: true,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username})
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	results := s.diag.Run(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.audit.List(100, 0)
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Server) handleNotificationsList(w http.ResponseWriter, r *http.Request) {
	notifs, err := s.notifs.ListUnresolved()
	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, "DB_UNAVAILABLE", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifs})
}
