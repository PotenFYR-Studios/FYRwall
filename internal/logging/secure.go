// Package logging provides secure, bounded log files for both the server
// and the agent (spec sections 11, 82, and the security requirements of
// section 35).
//
// Security model: log files contain operational detail that only the web
// administrator should read. Files are therefore written with 0640
// permissions owned root:fyrwall-equivalent (the service group), and the
// agent additionally encrypts its log with an AES-256-GCM key stored
// 0600 in the config directory. The key never leaves the host; the web
// admin reads logs through the authenticated API, which decrypts with the
// same key, so plaintext never sits world-readable on disk.
//
// Retention: a Rotator bounds logs by size (rotates at MaxBytes, keeps
// MaxBackups files) and by age (deletes files older than MaxAgeDays).
// A Janitor pass runs on every rotation and on a configurable interval,
// so neither the server nor the agent can fill the disk (spec section 82:
// never let telemetry consume all disk space).
package logging

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RetentionPolicy bounds log storage on disk.
type RetentionPolicy struct {
	MaxBytes    int64         // rotate when the active file exceeds this
	MaxBackups  int           // keep at most this many rotated files
	MaxAgeDays  int           // delete rotated files older than this
	Compression bool          // future: gzip rotated files; deletion-only for now
	Interval    time.Duration // janitor pass interval (0 = on rotation only)
}

// DefaultPolicy: 50 MiB active, 5 backups, 30 days - roughly 300 MiB
// ceiling per component, well under the disk-space warning threshold.
func DefaultPolicy() RetentionPolicy {
	return RetentionPolicy{
		MaxBytes:   50 << 20,
		MaxBackups: 5,
		MaxAgeDays: 30,
		Interval:   time.Hour,
	}
}

// Rotator is an io.WriteCloser implementing size and age retention.
type Rotator struct {
	mu      sync.Mutex
	path    string
	file    *os.File
	size    int64
	policy  RetentionPolicy
	stopJan chan struct{}
	janDone chan struct{}
}

// NewRotator opens (or creates) the log at path with 0640 permissions and
// starts the retention janitor if policy.Interval > 0.
func NewRotator(path string, policy RetentionPolicy) (*Rotator, error) {
	if path == "" {
		return nil, fmt.Errorf("empty log path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	fi, _ := f.Stat()
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}
	r := &Rotator{path: path, file: f, size: size, policy: policy}
	if policy.Interval > 0 {
		r.stopJan = make(chan struct{})
		r.janDone = make(chan struct{})
		go r.janitor()
	}
	return r, nil
}

// Write appends p, rotating first when the active file is over budget.
func (r *Rotator) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return 0, fmt.Errorf("rotator closed")
	}
	if r.size+int64(len(p)) > r.policy.MaxBytes {
		r.rotateLocked()
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// Close stops the janitor and releases the file.
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopJan != nil {
		close(r.stopJan)
		<-r.janDone
		r.stopJan = nil
	}
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// rotateLocked renames the active file with a timestamp suffix and starts
// a fresh one. Caller holds the lock.
func (r *Rotator) rotateLocked() {
	r.file.Close()
	r.file = nil
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	os.Rename(r.path, r.path+"."+stamp)
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return // writes will fail until Close/NewRotator; nothing else to do
	}
	r.file = f
	r.size = 0
	r.enforceLocked()
}

// enforceLocked applies MaxBackups and MaxAgeDays to rotated files.
func (r *Rotator) enforceLocked() {
	dir := filepath.Dir(r.path)
	base := filepath.Base(r.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type rotated struct {
		name string
		mod  time.Time
	}
	var olds []rotated
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), base+".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		olds = append(olds, rotated{e.Name(), info.ModTime()})
	}
	// Oldest first.
	sort.Slice(olds, func(i, j int) bool { return olds[i].mod.Before(olds[j].mod) })
	cutoff := time.Now().Add(-time.Duration(r.policy.MaxAgeDays) * 24 * time.Hour)
	keep := len(olds)
	for _, o := range olds {
		tooMany := keep > r.policy.MaxBackups
		tooOld := o.mod.Before(cutoff)
		if tooMany || tooOld {
			os.Remove(filepath.Join(dir, o.name))
			keep--
		}
	}
}

func (r *Rotator) janitor() {
	defer close(r.janDone)
	t := time.NewTicker(r.policy.Interval)
	defer t.Stop()
	for {
		select {
		case <-r.stopJan:
			return
		case <-t.C:
			r.mu.Lock()
			r.enforceLocked()
			r.mu.Unlock()
		}
	}
}

// --- Agent log encryption -----------------------------------------------

// keyFileName is where the log-encryption key lives (0600, next to the
// agent config). Losing the key loses the agent logs; that is the correct
// failure direction for confidential data.
const keyFileName = "agent-log.key"

// KeyFilePath returns the key location for a given config directory.
func KeyFilePath(configDir string) string {
	return filepath.Join(configDir, keyFileName)
}

// LoadOrCreateKey reads the 32-byte log-encryption key, creating a random
// one (0600) on first use. The key never leaves the host.
func LoadOrCreateKey(configDir string) ([]byte, error) {
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, err
	}
	kp := KeyFilePath(configDir)
	if b, err := os.ReadFile(kp); err == nil && len(b) == 32 {
		return b, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(kp, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptingRotator wraps a Rotator, writing AES-256-GCM ciphertext so the
// agent's on-disk logs are unreadable to anyone without the key. Records
// are length-prefixed: 4-byte big-endian length, then nonce+ciphertext.
type EncryptingRotator struct {
	*Rotator
	aead cipher.AEAD
	mu   sync.Mutex
	seq  uint64
}

// NewEncryptingRotator builds an encrypted, retention-bounded log.
func NewEncryptingRotator(path string, key []byte, policy RetentionPolicy) (*EncryptingRotator, error) {
	rot, err := NewRotator(path, policy)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		rot.Close()
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		rot.Close()
		return nil, err
	}
	return &EncryptingRotator{Rotator: rot, aead: aead}, nil
}

// Write encrypts one record and appends it.
func (e *EncryptingRotator) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	nonce := make([]byte, e.aead.NonceSize())
	binary.BigEndian.PutUint64(nonce[:8], e.seq)
	e.seq++
	sealed := e.aead.Seal(nonce, nonce, p, nil)
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(sealed)))
	buf := append(hdr[:], sealed...)
	if _, err := e.Rotator.Write(buf); err != nil {
		return 0, err
	}
	return len(p), nil
}

// DecryptFile decrypts a complete encrypted log into plaintext. Used by
// the log API path on the server side of an agent connection, or locally
// by `fyrwall agent logs`.
func DecryptFile(path string, key []byte, maxBytes int64) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		raw = raw[:maxBytes] // bounded read; caller sees truncated output marker downstream
	}
	var out []byte
	for off := 0; off+4 <= len(raw); {
		n := binary.BigEndian.Uint32(raw[off : off+4])
		off += 4
		if n == 0 || off+int(n) > len(raw) {
			break // torn or corrupt tail: keep what we have
		}
		rec := raw[off : off+int(n)]
		off += int(n)
		if len(rec) < aead.NonceSize() {
			continue
		}
		plain, err := aead.Open(nil, rec[:aead.NonceSize()], rec[aead.NonceSize():], nil)
		if err != nil {
			continue // skip records that fail authentication
		}
		out = append(out, plain...)
	}
	return out, nil
}

// ensure io stays imported if future compression lands
var _ io.Reader = nil
