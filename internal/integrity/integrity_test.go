package integrity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPaths(t *testing.T) (bin, rec, cfg string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "fyrwall")
	rec = filepath.Join(dir, "integrity.json")
	cfg = filepath.Join(dir, "cfg")
	if err := os.WriteFile(bin, []byte("fake-binary-content-v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, rec, cfg
}

func TestRecordThenVerifyPristine(t *testing.T) {
	bin, rec, cfg := testPaths(t)
	if err := Record(bin, rec, cfg, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	if err := Verify(bin, rec, cfg); err != nil {
		t.Fatalf("pristine binary must verify: %v", err)
	}
}

func TestDetectsBinaryModification(t *testing.T) {
	bin, rec, cfg := testPaths(t)
	if err := Record(bin, rec, cfg, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	// Someone tampers with the binary after install.
	if err := os.WriteFile(bin, []byte("fake-binary-content-BACKDOORED"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Verify(bin, rec, cfg)
	if err == nil {
		t.Fatal("modified binary must fail verification")
	}
	if !strings.Contains(err.Error(), "integrity check failed") {
		t.Fatalf("expected tamper error, got: %v", err)
	}
}

func TestDetectsRecordTampering(t *testing.T) {
	bin, rec, cfg := testPaths(t)
	if err := Record(bin, rec, cfg, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	// Attacker edits the reference file to match a new binary but has no
	// host key: the MAC check must fail.
	if err := os.WriteFile(bin, []byte("backdoored-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Recompute what the record would need to be WITHOUT the key by
	// recording a different binary under a fresh key, then swapping the
	// record file in. MAC was computed under cfg's key; the swap keeps
	// the same key, so craft a fake record manually.
	fake := `{"binary_sha256":"` + strings.Repeat("ab", 32) + `","recorded_at":"now","version":"0.1.0","mac":"deadbeef"}`
	if err := os.WriteFile(rec, []byte(fake), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Verify(bin, rec, cfg); err == nil {
		t.Fatal("forged record must fail MAC verification")
	}
}

func TestMissingRecordIsNotTamper(t *testing.T) {
	bin, rec, cfg := testPaths(t)
	err := Verify(bin, rec, cfg)
	if err == nil {
		t.Fatal("missing record must error")
	}
	if strings.Contains(err.Error(), "integrity check failed") {
		t.Fatalf("missing record is not tamper: %v", err)
	}
}

func TestKeyFilePerms(t *testing.T) {
	_, _, cfg := testPaths(t)
	keyPath := filepath.Join(cfg, "integrity.key")
	if _, err := hostKey(cfg); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("integrity key perms = %o, want 600", fi.Mode().Perm())
	}
}

func TestReRecordAfterVerifiedUpdate(t *testing.T) {
	bin, rec, cfg := testPaths(t)
	if err := Record(bin, rec, cfg, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	// Simulate a verified update replacing the binary.
	if err := os.WriteFile(bin, []byte("fake-binary-content-v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Verify(bin, rec, cfg); err == nil {
		t.Fatal("old record must reject the new binary")
	}
	// The updater re-records after verification.
	if err := Record(bin, rec, cfg, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := Verify(bin, rec, cfg); err != nil {
		t.Fatalf("re-recorded binary must verify: %v", err)
	}
}
