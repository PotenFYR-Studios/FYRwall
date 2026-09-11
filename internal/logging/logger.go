// Package logging provides structured zerolog logging with secret
// redaction (spec sections 11, 35 and 81). Never log passwords, tokens, keys.
package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Redacted placeholder used whenever a sensitive value would be logged.
const Redacted = "[REDACTED]"

// Keys that must never appear in log output in plaintext form.
var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private_key", "session", "cookie", "authorization", "dsn",
	"credential", "ssh_key", "passphrase", "cert_key",
}

// New builds the root logger. writes to stderr; fileOut (optional) is the
// already-open rotating file writer.
func New(level, format string, fileOut io.Writer) zerolog.Logger {
	var w io.Writer = os.Stderr
	if fileOut != nil {
		w = io.MultiWriter(os.Stderr, fileOut)
	}
	var l zerolog.Logger
	switch strings.ToLower(format) {
	case "console":
		l = zerolog.New(zerolog.ConsoleWriter{Out: w, TimeFormat: time.RFC3339})
	default:
		l = zerolog.New(w)
	}
	l = l.With().Timestamp().Caller().Logger()
	switch strings.ToLower(level) {
	case "debug":
		l = l.Level(zerolog.DebugLevel)
	case "warn":
		l = l.Level(zerolog.WarnLevel)
	case "error":
		l = l.Level(zerolog.ErrorLevel)
	case "fatal":
		l = l.Level(zerolog.FatalLevel)
	default:
		l = l.Level(zerolog.InfoLevel)
	}
	return l
}

// SanitizeMap returns a copy of m with any sensitive-key values redacted.
// Used before dumping config/params into logs, audit payloads, or
// diagnostics bundles.
func SanitizeMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = sanitizeValue(k, v)
	}
	return out
}

func sanitizeValue(key string, v any) any {
	lk := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lk, s) {
			return Redacted
		}
	}
	switch tv := v.(type) {
	case map[string]any:
		return SanitizeMap(tv)
	case []any:
		out := make([]any, len(tv))
		for i, e := range tv {
			out[i] = sanitizeValue(key, e)
		}
		return out
	default:
		return v
	}
}

// OpenLogFile opens (and creates parents for) the configured log file with
// 0640 permissions. Returns nil if path is empty or open fails - callers
// log the failure separately and continue with stderr-only logging.
func OpenLogFile(path string) io.Writer {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil
	}
	return f
}
