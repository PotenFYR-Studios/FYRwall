package app

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/health"
	"github.com/PotenFYR-Studios/FYRwall/internal/integrity"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

// netListener abstracts TCP/TLS listeners for the app wiring.
type netListener interface {
	Addr() net.Addr
	Close() error
	Accept() (net.Conn, error)
}

// listen binds the configured address/port, with TLS when enabled.
func listen(bind string, port int, tlsCfg config.TLSConfig) (netListener, error) {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("bind %s: %w", addr, err)
	}
	if tlsCfg.Enabled {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			ln.Close()
			return nil, fmt.Errorf("tls load: %w", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		if tlsCfg.ClientCAFile != "" {
			if caPEM, readErr := os.ReadFile(tlsCfg.ClientCAFile); readErr == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsConfig.ClientCAs = pool
					tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
				}
			}
		}
		return tls.NewListener(ln, tlsConfig), nil
	}
	return ln, nil
}

// httpServer is a minimal runner around http.Server with timeouts
// (slow-client defense, spec section 77).
type httpServer struct {
	ln      netListener
	handler http.Handler
}

func (h *httpServer) serve() error {
	srv := &http.Server{
		Handler:           h.handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return srv.Serve(h.ln.(net.Listener))
}

// recoveryHandler serves a minimal health page when the DB is unavailable
// (spec sections 8.7, 44: web recovery page stays available when safe).
func recoveryHandler(dbErr error, ht *health.Tracker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/system/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"state":"BLOCKED","error":"database unavailable"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "FYRwall is in BLOCKED state.\nDatabase error: %v\nFix the database and restart fyrwall-server.\n", dbErr)
	})
	return mux
}

// restoreList prints stored restore points with pagination guard.
func restoreList(db *database.DB) error {
	rows, err := db.SQL().Query(`SELECT id, kind, reason, backend, state_hash, created_at
		FROM restore_points ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, kind, backend, stateHash, created string
		var reason sql.NullString
		if err := rows.Scan(&id, &kind, &reason, &backend, &stateHash, &created); err != nil {
			return err
		}
		fmt.Printf("%s  %-10s %-10s %-14s %s\n", created, kind, backend, stateHash[:12], reason.String)
	}
	return rows.Err()
}

// selfIntegrityCheck verifies the running executable against the
// integrity record on boot (spec: tamper protection for server, web GUI
// and agent). Missing record is tolerated on first run (auto-record);
// a mismatch is a hard failure surfaced to the caller.
func selfIntegrityCheck(configDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, _ = filepathAbs(exe)
	rec := filepathJoin(configDir, "integrity.json")
	if err := integrity.Verify(exe, rec, configDir); err != nil {
		// First boot: no record yet, so create one and pass.
		if strings.Contains(err.Error(), "no integrity record") {
			return integrity.Record(exe, rec, configDir, version.Version)
		}
		return err
	}
	return nil
}

func filepathAbs(p string) (string, error) { return filepath.Abs(p) }

func filepathJoin(elem ...string) string { return filepath.Join(elem...) }

func filepathDir(p string) string { return filepath.Dir(p) }
