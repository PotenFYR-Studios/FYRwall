package auth

import (
	"strings"
	"testing"
	"time"
)

func timeNowMinusMinute() time.Time { return time.Now().Add(-time.Minute) }
func timeNowPlusMinute() time.Time  { return time.Now().Add(time.Minute) }

func TestHashVerifyRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple", DefaultArgonParams())
	if err != nil {
		t.Fatal(err)
	}
	ok, rehash, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	if rehash {
		t.Error("fresh hash should not need rehash")
	}
	ok, _, _ = VerifyPassword("wrong password entirely", hash)
	if ok {
		t.Fatal("wrong password must not verify")
	}
}

func TestVerifyMalformedHash(t *testing.T) {
	ok, _, err := VerifyPassword("x", "not-a-phc-string")
	if err == nil || ok {
		t.Fatal("expected error on malformed hash")
	}
	ok, _, err = VerifyPassword("x", "$argon2id$v=19$m=64,t=1,p=2$ab$cd")
	if err == nil && ok {
		t.Fatal("expected failure or error on truncated hash")
	}
}

func TestHashIsSalted(t *testing.T) {
	h1, _ := HashPassword("same-password", DefaultArgonParams())
	h2, _ := HashPassword("same-password", DefaultArgonParams())
	if h1 == h2 {
		t.Fatal("two hashes of the same password must differ (salt)")
	}
}

func TestHashFormatIsArgon2idPHC(t *testing.T) {
	h, _ := HashPassword("x", DefaultArgonParams())
	if !strings.HasPrefix(h, "$argon2id$v=") {
		t.Fatalf("hash = %q, want PHC argon2id format", h)
	}
}

func TestRBAC(t *testing.T) {
	if !HasPermission(RoleSuperAdmin, PermUsersManage) {
		t.Error("super admin must have all permissions")
	}
	if !HasPermission(RoleAdmin, PermFirewallApply) {
		t.Error("admin must apply firewall")
	}
	if HasPermission(RoleViewer, PermFirewallWrite) {
		t.Error("viewer must not write firewall")
	}
	if HasPermission(RoleViewer, PermUsersManage) {
		t.Error("viewer must not manage users")
	}
	if HasPermission(RoleAuditor, PermFirewallApply) {
		t.Error("auditor is read-only")
	}
	if !HasPermission(RoleAuditor, PermAuditRead) {
		t.Error("auditor must read audit log")
	}
	if ValidRole("hacker") {
		t.Error("unknown role must be invalid")
	}
}

func TestRateLimiter(t *testing.T) {
	// Small per-minute budget: first attempts pass, then blocked.
	l := NewLoginRateLimiter(2)
	ip := "10.9.9.9"
	allowed := 0
	for i := 0; i < 10; i++ {
		if l.Allow(ip) {
			allowed++
		}
	}
	if allowed > 3 {
		t.Fatalf("rate limiter allowed %d/10 attempts with budget 2/min", allowed)
	}
	// Different IP unaffected.
	if !l.Allow("10.9.9.10") {
		t.Fatal("second IP should not be throttled by first IP's budget")
	}
}

func TestNormalizeIP(t *testing.T) {
	if got := NormalizeIP("192.168.1.5:8080"); got != "192.168.1.5" {
		t.Errorf("NormalizeIP = %q", got)
	}
	if got := NormalizeIP("[::1]:80"); got != "::1" {
		t.Errorf("NormalizeIP v6 = %q", got)
	}
}

func TestSessionRevocation(t *testing.T) {
	RevokeUserSessions("bob")
	if !SessionRevokedAfter("bob", timeNowMinusMinute()) {
		t.Error("sessions logged in before revocation must be revoked")
	}
	if SessionRevokedAfter("bob", timeNowPlusMinute()) {
		t.Error("sessions logged in after revocation must survive")
	}
	if SessionRevokedAfter("alice", timeNowMinusMinute()) {
		t.Error("unrelated user must be unaffected")
	}
}
