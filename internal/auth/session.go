package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
)

// SessionManager wraps scs with secure cookie defaults (spec section 13.1).
type SessionManager struct {
	scs *scs.SessionManager
}

// NewSessionManager builds a server-side session store. cookieSecure must
// be true whenever the UI is served over HTTPS.
func NewSessionManager(idleTimeout time.Duration, cookieSecure bool) *SessionManager {
	m := scs.New()
	m.Lifetime = 24 * time.Hour
	m.IdleTimeout = idleTimeout
	m.Cookie.Name = "fyrwall_session"
	m.Cookie.HttpOnly = true
	m.Cookie.Secure = cookieSecure
	m.Cookie.SameSite = http.SameSiteStrictMode
	m.Cookie.Persist = false
	return &SessionManager{scs: m}
}

// Manager exposes the underlying scs manager for middleware wiring.
func (s *SessionManager) Manager() *scs.SessionManager { return s.scs }

// CSRF token header name and session key.
const (
	CSRFHeader  = "X-FYRwall-CSRF"
	CSRFSession = "csrf_token"
)

// EnsureCSRF returns the session's CSRF token, creating one if absent.
// The token is random per session and compared in constant time.
func (s *SessionManager) EnsureCSRF(r *http.Request) string {
	tok := s.scs.GetString(r.Context(), CSRFSession)
	if tok == "" {
		tok = randomToken(32)
		s.scs.Put(r.Context(), CSRFSession, tok)
	}
	return tok
}

// CheckCSRF validates the CSRF token on state-changing requests.
func (s *SessionManager) CheckCSRF(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	sent := r.Header.Get(CSRFHeader)
	if sent == "" {
		// Also accept the standard header name for proxy-friendly setups.
		sent = r.Header.Get("X-CSRF-Token")
	}
	want := s.scs.GetString(r.Context(), CSRFSession)
	if want == "" || sent == "" {
		return false
	}
	return secureEqual(sent, want)
}

// secureEqual is a constant-time string comparison.
func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// revocation tracks session revocation for user management features
// (revoke sessions on password reset / user disable).
type revocation struct {
	mu       sync.Mutex
	revokedU map[string]time.Time // username -> revocation instant
}

var globalRevocation = &revocation{revokedU: map[string]time.Time{}}

// RevokeUserSessions marks all sessions of user as invalid from now.
func RevokeUserSessions(username string) {
	globalRevocation.mu.Lock()
	defer globalRevocation.mu.Unlock()
	globalRevocation.revokedU[username] = time.Now()
}

// SessionRevokedAfter reports whether the user's sessions were revoked
// after the given login time.
func SessionRevokedAfter(username string, loginAt time.Time) bool {
	globalRevocation.mu.Lock()
	defer globalRevocation.mu.Unlock()
	t, ok := globalRevocation.revokedU[username]
	return ok && loginAt.Before(t)
}

func randomToken(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	fill := make([]byte, n)
	// crypto/rand-backed via scs's token generator alternative: simple
	// rejection-free approach with CryptoRand.
	readRandom(fill)
	for i := range b {
		b[i] = charset[int(fill[i])%len(charset)]
	}
	return strings.TrimSpace(string(b))
}
