package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/diagnostics"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/health"
)

// stubBackend satisfies firewall.FirewallBackend without touching the host.
type stubBackend struct{}

func (stubBackend) Name() string { return "stub" }
func (stubBackend) Detect(_ context.Context) firewall.DetectionResult {
	return firewall.DetectionResult{Name: "stub", Found: true}
}
func (stubBackend) Status(_ context.Context) (firewall.FirewallStatus, error) {
	return firewall.FirewallStatus{Backend: "stub", Enabled: true, CheckedAt: time.Now()}, nil
}
func (stubBackend) ListRules(_ context.Context) ([]firewall.Rule, error) {
	return []firewall.Rule{}, nil
}
func (stubBackend) Validate(_ context.Context, tx firewall.Transaction) firewall.ValidationResult {
	return firewall.ValidateTx(tx, nil)
}
func (stubBackend) Snapshot(_ context.Context, reason string) (firewall.Snapshot, error) {
	return firewall.Snapshot{ID: "snap-test", Reason: reason, StateHash: "abc"}, nil
}
func (stubBackend) Apply(_ context.Context, tx firewall.Transaction) (firewall.ApplyResult, error) {
	return firewall.ApplyResult{TxnID: tx.ID, Applied: len(tx.Actions), StateHash: "abc"}, nil
}
func (stubBackend) Verify(_ context.Context, exp firewall.StateHash) (firewall.VerifyResult, error) {
	return firewall.VerifyResult{Match: string(exp) == "abc", Observed: "abc", Expected: string(exp)}, nil
}
func (stubBackend) Restore(_ context.Context, _ firewall.Snapshot) error { return nil }
func (stubBackend) Reload(_ context.Context) error                       { return nil }
func (stubBackend) Restart(_ context.Context) error                      { return nil }

func testLogger() zerolog.Logger { return zerolog.Nop() }

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Database.SQLitePath = filepath.Join(dir, "test.db")
	db, err := database.OpenSQLite(cfg.Database.SQLitePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(database.Schema()); err != nil {
		t.Fatal(err)
	}

	// Bootstrap an admin for auth flows.
	hash, _ := auth.HashPassword("bootstrap-password-1", auth.DefaultArgonParams())
	users := database.NewUserRepo(db)
	if _, err := users.Create("admin", hash, auth.RoleSuperAdmin, false); err != nil {
		t.Fatal(err)
	}

	mgr := firewall.NewManager(stubBackend{}, nil, nil)
	mgr.SetOwnership(firewall.OwnershipReport{Owner: firewall.OwnerUFW, WritesAllowed: true})
	ht := health.NewTracker()
	srv := New(cfg, testLogger(), db, mgr, ht, diagnostics.NewRegistry())
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts
}

func loginAndGetCSRF(t *testing.T, ts *httptest.Server) (*http.Client, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "bootstrap-password-1"})
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("login failed: %d", resp.StatusCode)
	}
	// Session cookie lives in the default client jar; reuse the login
	// response's cookie jar via a shared client.
	jar, _ := cookieJarNew()
	jar.SetCookies(mustParse(ts.URL), resp.Cookies())
	client := &http.Client{Jar: jar}
	me, err := client.Get(ts.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer me.Body.Close()
	var out struct {
		CSRFToken string `json:"csrf_token"`
	}
	json.NewDecoder(me.Body).Decode(&out)
	return client, out.CSRFToken
}

func TestAnonymousCannotAccessAPI(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/firewall/rules")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for anonymous, got %d", resp.StatusCode)
	}
}

func TestLoginSuccessAndFailure(t *testing.T) {
	ts := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	resp, _ := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 on wrong password, got %d", resp.StatusCode)
	}
	_, csrf := loginAndGetCSRF(t, ts)
	if csrf == "" {
		t.Fatal("expected CSRF token after login")
	}
}

func TestStrictJSONRejected(t *testing.T) {
	ts := newTestServer(t)
	client, csrf := loginAndGetCSRF(t, ts)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/firewall/transactions",
		bytes.NewBufferString(`{"unknown_field": true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FYRwall-CSRF", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on unknown field, got %d", resp.StatusCode)
	}
}

func TestSecurityHeaders(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff header")
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" || !bytes.Contains([]byte(csp), []byte("frame-ancestors 'none'")) {
		t.Errorf("weak CSP: %q", csp)
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options DENY")
	}
}

func TestVersionEndpoint(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var v map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	if v["version"] == "" {
		t.Fatal("version field missing")
	}
}

func cookieJarNew() (*cookiejar.Jar, error) { return cookiejar.New(nil) }

func mustParse(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}
