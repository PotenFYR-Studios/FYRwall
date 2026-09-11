// Package auth implements local authentication: Argon2id password hashing,
// server-side sessions, CSRF protection, RBAC and login rate limiting
// (spec sections 13, 35).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. Defaults are conservative; benchmarked values can be
// configured per host (spec section 13.1).
type ArgonParams struct {
	Time    uint32
	Memory  uint32 // KiB
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// DefaultArgonParams are sized for a small VPS (~64 MiB, ~1 pass).
func DefaultArgonParams() ArgonParams {
	return ArgonParams{Time: 1, Memory: 64 * 1024, Threads: 2, KeyLen: 32, SaltLen: 16}
}

// HashPassword derives an Argon2id hash in the PHC string format.
func HashPassword(password string, p ArgonParams) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword checks a password against a PHC-format Argon2id hash in
// constant time. Supports rehash detection: ok=true, needsRehash=true when
// the hash used weaker params than current policy.
func VerifyPassword(password, phc string) (bool, bool, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, false, errors.New("malformed password hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, false, err
	}
	var mem, timeP uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &timeP, &threads); err != nil {
		return false, false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, false, err
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, timeP, mem, threads, uint32(len(key)))
	if subtle.ConstantTimeCompare(candidate, key) == 1 {
		cur := DefaultArgonParams()
		rehash := mem != cur.Memory || timeP != cur.Time || threads != cur.Threads
		return true, rehash, nil
	}
	return false, false, nil
}
