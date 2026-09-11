package secureconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("server:\n  port: 7443\nsecurity:\n  password_hash: supersecret\n")
	if err := s.Save(plain); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("round trip mismatch:\n%s\nvs\n%s", got, plain)
	}
}

func TestOnDiskFileIsEncryptedNotPlaintext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s, _ := New(path)
	_ = s.Save([]byte("token: abc123"))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 7 && string(raw[:7]) == "token:" {
		t.Fatal("plaintext secret found on disk")
	}
	if !strings.HasPrefix(string(raw), "FYRCFG1") {
		t.Fatalf("expected magic prefix, got %q", raw[:10])
	}
	// The secret must not appear anywhere in the raw bytes.
	if strings.Contains(string(raw), "abc123") {
		t.Fatal("secret recoverable from raw file bytes")
	}
}

func TestKeyFilePermissions(t *testing.T) {
	dir := t.TempDir()
	kp := filepath.Join(dir, KeyFile)
	if _, err := loadOrCreateKey(kp); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(kp)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("keyfile perms = %o, want 600", fi.Mode().Perm())
	}
	// Deterministic: second load returns the same key.
	k1, _ := loadOrCreateKey(kp)
	k2, _ := loadOrCreateKey(kp)
	if string(k1) != string(k2) {
		t.Fatal("key changed between loads")
	}
}

func TestConfigFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s, _ := New(path)
	if err := s.Save([]byte("a: 1")); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("config perms = %o, want 640", fi.Mode().Perm())
	}
}

func TestTamperDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s, _ := New(path)
	_ = s.Save([]byte("secret: value"))

	// Flip one byte in the ciphertext.
	raw, _ := os.ReadFile(path)
	raw[len(raw)-3] ^= 0xFF
	os.WriteFile(path, raw, 0o640)

	if err := s.VerifyIntegrity(); err == nil {
		t.Fatal("tampered config must fail integrity verification")
	}
}

func TestWrongKeyDetected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s, _ := New(path)
	_ = s.Save([]byte("a: 1"))

	// Simulate keyfile loss/replacement: new key generated, then a fresh
	// store (as after a restart) must fail to decrypt the old config.
	os.Remove(filepath.Join(dir, KeyFile))
	if _, err := loadOrCreateKey(filepath.Join(dir, KeyFile)); err != nil {
		t.Fatal(err)
	}
	s2, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s2.VerifyIntegrity(); err == nil {
		t.Fatal("different key must fail authentication (old config unreadable)")
	}
}

func TestPlaintextMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	legacy := []byte("server:\n  port: 7443\n")
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Load() // triggers migration
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(legacy) {
		t.Fatal("migrated content mismatch")
	}

	// On-disk file is now encrypted.
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "port: 7443") {
		t.Fatal("plaintext survived migration")
	}
	// Original preserved for review.
	if _, err := os.Stat(path + ImportedSuffix); err != nil {
		t.Fatal("imported plaintext backup missing")
	}
}

func TestSaveEmptyRejected(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(filepath.Join(dir, "config.yaml"))
	if err := s.Save(nil); err == nil {
		t.Fatal("empty config must be rejected")
	}
}
