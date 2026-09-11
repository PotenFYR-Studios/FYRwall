// Secure config persistence: config files on disk are AES-256-GCM
// encrypted with a key that lives only in a root-owned keyfile (0600,
// root:root). The plaintext config never needs to exist on disk with
// wide permissions; the server reads it through this package and the
// ONLY supported write path is an authenticated web-GUI settings change
// (or a root-run CLI wizard on the host).
//
// Threat model addressed:
//   - another user or service on the box reading secrets from config
//   - tampered config (GCM authentication detects any modification)
//   - accidental plaintext copies in backups (backups of the encrypted
//     file stay encrypted; the keyfile is excluded from backups)
//
// Migration: a plaintext config.yaml found at load time is imported
// once, encrypted in place, and the plaintext file is renamed to
// config.yaml.imported (kept briefly, 0600, for review) then the
// operator deletes it - or --purge removes it immediately.
package secureconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// KeyFile holds the 32-byte config encryption key. root:root 0600.
	KeyFile = "config.key"
	// Magic prefixes an encrypted config file.
	Magic = "FYRCFG1"
	// ImportedSuffix marks a plaintext config preserved after migration.
	ImportedSuffix = ".imported"
)

// Store binds a config path to its keyfile.
type Store struct {
	Path    string // e.g. /etc/fyrwall/config.yaml
	KeyPath string // e.g. /etc/fyrwall/config.key
	key     []byte
}

// New opens (creating the key on first use) a secure store for path.
// The keyfile is created 0600 root:root; the config file 0640 root:fyrwall
// so the unprivileged service group can read but not write it.
func New(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	s := &Store{Path: path, KeyPath: filepath.Join(dir, KeyFile)}
	key, err := loadOrCreateKey(s.KeyPath)
	if err != nil {
		return nil, err
	}
	s.key = key
	return s, nil
}

func loadOrCreateKey(kp string) ([]byte, error) {
	if b, err := os.ReadFile(kp); err == nil {
		if len(b) == 32 {
			return b, nil
		}
		return nil, fmt.Errorf("keyfile %s has wrong size %d (want 32)", kp, len(b))
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(kp, key, 0o600); err != nil {
		return nil, err
	}
	// Best-effort tight ownership; may fail when not root, which is fine
	// for dev/test (file is still 0600).
	_ = os.Chown(kp, 0, 0)
	return key, nil
}

func (s *Store) aead() (cipher.AEAD, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Load returns the decrypted config bytes. If the file does not exist it
// returns nil, nil (fresh install). If the file is plaintext (legacy),
// it migrates: encrypt in place and preserve the original as
// config.yaml.imported with 0600 for operator review.
func (s *Store) Load() ([]byte, error) {
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if isEncrypted(raw) {
		return s.decrypt(raw)
	}
	// Legacy plaintext: migrate.
	if err := s.saveEncrypted(raw); err != nil {
		return nil, fmt.Errorf("migrating plaintext config: %w", err)
	}
	_ = os.WriteFile(s.Path+ImportedSuffix, raw, 0o600)
	return raw, nil
}

// Save encrypts and atomically writes the config. Called ONLY from the
// web GUI settings handler (or the root CLI wizard) - there is no other
// supported write path, matching the "config is GUI-only" policy.
func (s *Store) Save(plaintext []byte) error {
	if len(plaintext) == 0 {
		return errors.New("refusing to save empty config")
	}
	return s.saveEncrypted(plaintext)
}

func (s *Store) saveEncrypted(plaintext []byte) error {
	aead, err := s.aead()
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := aead.Seal(nonce, nonce, plaintext, nil)
	out := make([]byte, 0, len(Magic)+4+len(sealed))
	out = append(out, Magic...)
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(sealed)))
	out = append(out, l[:]...)
	out = append(out, sealed...)

	// Atomic write: temp in same dir, fsync, rename, 0640 root-group.
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o640); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chown(tmpName, 0, -1) // best-effort root owner when run as root
	return os.Rename(tmpName, s.Path)
}

func (s *Store) decrypt(raw []byte) ([]byte, error) {
	aead, err := s.aead()
	if err != nil {
		return nil, err
	}
	if len(raw) < len(Magic)+4+aead.NonceSize() {
		return nil, errors.New("encrypted config truncated")
	}
	body := raw[len(Magic):]
	n := binary.BigEndian.Uint32(body[:4])
	body = body[4:]
	if int(n) > len(body) {
		return nil, errors.New("encrypted config length mismatch")
	}
	return aead.Open(nil, body[:aead.NonceSize()], body[aead.NonceSize():n], nil)
}

func isEncrypted(raw []byte) bool {
	return len(raw) > len(Magic) && string(raw[:len(Magic)]) == Magic
}

// VerifyIntegrity reports whether the on-disk config decrypts cleanly -
// used by preflight to detect tampering or keyfile loss (wrong key =>
// GCM auth failure => BLOCKED health with a clear remediation).
func (s *Store) VerifyIntegrity() error {
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to verify
		}
		return err
	}
	if !isEncrypted(raw) {
		return errors.New("config file is plaintext; restart once to migrate or secure it manually")
	}
	_, err = s.decrypt(raw)
	return err
}
