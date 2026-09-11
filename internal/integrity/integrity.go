// Package integrity implements binary tamper protection: at install time
// (and on every version change) FYRwall records the SHA-256 hash of its
// own executable in a protected reference file. On every boot the agent
// and server re-hash the running binary and compare against the
// reference. A mismatch means the binary was modified, replaced, or
// corrupted after install - FYRwall refuses privileged operations and
// raises a critical issue instead of running possibly compromised code.
//
// Trust model: the reference file lives with the config/key material
// (root-owned, 0640) and itself stores the hash of the reference under
// a keyed MAC so an attacker who replaces BOTH the binary and the
// reference file cannot produce a consistent pair without the host key.
// This is defense in depth, not a substitute for signed packages;
// release tarballs carry SHA256SUMS for the initial trust anchor.
package integrity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Reference is the on-disk record of the binary hash (JSON).
type Reference struct {
	BinarySHA256 string `json:"binary_sha256"`
	RecordedAt   string `json:"recorded_at"`
	Version      string `json:"version"`
	MAC          string `json:"mac"` // HMAC-SHA256(hostKey, binary hash + version)
}

// ErrTampered is the sentinel returned when the running binary does not
// match its recorded hash.
var ErrTampered = errors.New("binary integrity check failed: executable changed since install")

// hostKey loads or creates the integrity key (32 bytes, 0600) in
// configDir. Distinct from the log-encryption key to keep concerns
// separable; both are host-local secrets.
func hostKey(configDir string) ([]byte, error) {
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, err
	}
	kp := filepath.Join(configDir, "integrity.key")
	if b, err := os.ReadFile(kp); err == nil {
		if len(b) == 32 {
			return b, nil
		}
		return nil, fmt.Errorf("integrity key %s wrong size %d", kp, len(b))
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(kp, key, 0o600); err != nil {
		return nil, err
	}
	_ = os.Chown(kp, 0, 0)
	return key, nil
}

// Record hashes executablePath and writes the reference file at
// recordPath. Called at install and after verified updates.
func Record(executablePath, recordPath, configDir, version string) error {
	binHash, err := hashFile(executablePath)
	if err != nil {
		return fmt.Errorf("hash binary: %w", err)
	}
	key, err := hostKey(configDir)
	if err != nil {
		return err
	}
	mac := computeMAC(key, binHash, version)
	rec := Reference{
		BinarySHA256: hex.EncodeToString(binHash),
		RecordedAt:   nowUTC(),
		Version:      version,
		MAC:          mac,
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(recordPath, b, 0o640); err != nil {
		return err
	}
	_ = os.Chown(recordPath, 0, 0)
	return nil
}

// Verify re-hashes the running binary and checks it against the record.
// Returns nil when pristine, ErrTampered on mismatch, and a wrapped
// error when the record or key is missing (fresh install path: callers
// treat missing record as "record now" not "tamper").
func Verify(executablePath, recordPath, configDir string) error {
	raw, err := os.ReadFile(recordPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no integrity record at %s (run install or `fyrwall integrity record`)", recordPath)
		}
		return err
	}
	var rec Reference
	if err := json.Unmarshal(raw, &rec); err != nil {
		return fmt.Errorf("corrupt integrity record: %w", err)
	}
	binHash, err := hashFile(executablePath)
	if err != nil {
		return err
	}
	current := hex.EncodeToString(binHash)
	if current != rec.BinarySHA256 {
		return fmt.Errorf("%w: running %s, recorded %s", ErrTampered, current[:16], rec.BinarySHA256[:16])
	}
	// The reference itself must be authentic: MAC covers binary hash and
	// version under the host key.
	key, err := hostKey(configDir)
	if err != nil {
		return err
	}
	want := computeMAC(key, binHash, rec.Version)
	if !hmac.Equal([]byte(want), []byte(rec.MAC)) {
		return fmt.Errorf("%w: integrity record MAC mismatch (binary and record do not form a trusted pair)", ErrTampered)
	}
	return nil
}

func computeMAC(key []byte, binHash []byte, version string) string {
	m := hmac.New(sha256.New, key)
	m.Write(binHash)
	m.Write([]byte(version))
	return hex.EncodeToString(m.Sum(nil))
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func ioCopy(dst io.Writer, src io.Reader) (int64, error) { return io.Copy(dst, src) }

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := ioCopy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
