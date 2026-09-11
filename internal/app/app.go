// Package app wires the application together: startup, preflight, health
// propagation, server and agent lifecycles (spec sections 8, 9, 10.2).
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/agent"
	"github.com/PotenFYR-Studios/FYRwall/internal/api"
	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/diagnostics"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall/detectors"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall/iptables"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall/ufw"
	"github.com/PotenFYR-Studios/FYRwall/internal/health"
	"github.com/PotenFYR-Studios/FYRwall/internal/logging"
	"github.com/PotenFYR-Studios/FYRwall/internal/system"
	"github.com/PotenFYR-Studios/FYRwall/internal/ui"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

// buildLogger creates the root logger from config.
func buildLogger(cfg *config.Config) zerolog.Logger {
	return logging.New(cfg.Logging.Level, cfg.Logging.Format, logging.OpenLogFile(cfg.Logging.FilePath))
}

// openDB opens and migrates the database.
func openDB(cfg *config.Config) (*database.DB, error) {
	db, err := database.OpenSQLite(cfg.Database.SQLitePath)
	if err != nil {
		return nil, err
	}
	if err := db.IntegCheck(); err != nil {
		db.Close()
		return nil, fmt.Errorf("integrity: %w", err)
	}
	if err := db.Migrate(database.Schema()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// detectBackend picks and constructs the firewall backend per config and
// availability (spec section 6: capability detection, never assumptions).
func detectBackend(ctx context.Context, cfg *config.Config) (firewall.FirewallBackend, firewall.OwnershipReport, error) {
	var ufwBackend *ufw.Backend
	if u, err := ufw.New(); err == nil {
		ufwBackend = u
	}
	var iptBackend *iptables.Backend
	iptTools, iptErr := iptables.DetectTools(ctx)
	if iptErr == nil {
		iptBackend = iptables.New(iptTools)
	}

	ufwSvc := detectors.UFWResult{}
	if ufwBackend != nil {
		ufwSvc = detectors.DetectUFWService(ctx, ufwBackend.Detect(ctx).BinaryPath)
	}
	fw := detectors.DetectFirewalld(ctx)
	nft := detectors.DetectNftables(ctx)

	iptMode := "unknown"
	if iptErr == nil {
		iptMode = string(iptTools.Mode)
	}
	ownership := firewall.ResolveOwnership(ufwSvc.Active, fw.Running, nft.RulesetItems > 0, iptMode,
		cfg.Firewall.AllowWriteOnManagerConflict)

	// Backend selection honors explicit config first, then availability.
	var chosen firewall.FirewallBackend
	switch cfg.Firewall.Backend {
	case "ufw":
		if ufwBackend == nil {
			return nil, ownership, firewall.ErrBackendUnavailable
		}
		chosen = ufwBackend
	case "iptables":
		if iptBackend == nil {
			return nil, ownership, firewall.ErrBackendUnavailable
		}
		chosen = iptBackend
	default: // auto
		if ownership.Owner == firewall.OwnerUFW && ufwBackend != nil {
			chosen = ufwBackend
		} else if iptBackend != nil {
			chosen = iptBackend
		} else if ufwBackend != nil {
			chosen = ufwBackend
		}
	}
	if chosen == nil {
		return nil, ownership, firewall.ErrBackendUnavailable
	}
	return chosen, ownership, nil
}

// emitStartupIssue persists a startup problem as a deduplicated
// notification so it shows in logs, notifications and the web UI
// (spec sections 10.2, 54).
func emitStartupIssue(db *database.DB, code, severity, component, summary, remediation string) {
	fingerprint := code + "|" + component + "|host"
	repo := database.NewNotificationRepo(db)
	if err := repo.Upsert(database.Notification{
		Code: code, Severity: severity, Category: "startup", Component: component,
		Summary: summary, Remediation: remediation, Fingerprint: fingerprint,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "notification persist failed: %v\n", err)
	}
}

// RunServer runs the full server lifecycle (spec section 9 state machine).
func RunServer(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := buildLogger(cfg)
	ht := health.NewTracker()
	ht.Set("server", health.StateStarting, "")
	log.Info().Str("event_code", "STARTING").Msgf("fyrwall %s starting", version.Version)

	// Preflight: platform facts (read-only).
	plat := system.DetectPlatform()
	log.Info().Str("event_code", "PREFLIGHT_PLATFORM").
		Str("arch", plat.Arch).Str("kernel", plat.KernelVersion).
		Str("init", plat.InitSystem).Bool("container", plat.IsContainer).
		Msg("platform detected")

	// Database.
	db, err := openDB(cfg)
	if err != nil {
		ht.Set("database", health.StateBlocked, err.Error())
		emitStartupIssue(db, "DB_UNAVAILABLE", "critical", "database", err.Error(),
			"Check sqlite_path permissions and disk space, then restart.")
		serveRecoveryUI(cfg, log, ht, err)
		return err
	}
	defer db.Close()
	ht.Set("database", health.StateHealthy, "sqlite ready")

	// Binary integrity: server + embedded web GUI tamper check on boot
	// (spec: protect the whole stack, not just the agent).
	configDir := filepathDir(configPath)
	if ierr := selfIntegrityCheck(configDir); ierr != nil {
		ht.Set("integrity", health.StateDegraded, ierr.Error())
		log.Error().Str("event_code", "INTEGRITY_FAILED").Err(ierr).
			Msg("binary tamper check failed; privileged operations disabled")
		emitStartupIssue(db, "INTEGRITY_FAILED", "critical", "integrity",
			ierr.Error(), "Reinstall FYRwall from a verified release (SHA256SUMS) or restore the integrity record.")
	} else {
		ht.Set("integrity", health.StateHealthy, "binary hash verified")
	}

	// Firewall backend detection (read-only).
	backend, ownership, err := detectBackend(ctx, cfg)
	if err != nil {
		ht.Set("firewall", health.StateBlocked, err.Error())
		emitStartupIssue(db, "FW_BACKEND_NOT_FOUND", "critical", "firewall",
			"no usable firewall backend detected", "Install ufw or iptables, or adjust firewall.backend.")
	} else {
		if ownership.Owner == firewall.OwnerMultipleConflicting {
			ht.Set("firewall", health.StateDegraded, ownership.Reason)
			emitStartupIssue(db, "FW_BACKEND_CONFLICT", "critical", "firewall",
				ownership.Reason, "Disable all but one firewall manager, then acknowledge the conflict.")
		} else if !ownership.WritesAllowed {
			ht.Set("firewall", health.StateDegraded, ownership.Reason)
			emitStartupIssue(db, "FW_WRITES_BLOCKED", "warning", "firewall",
				ownership.Reason, "Review the firewall ownership report in the dashboard.")
		} else {
			ht.Set("firewall", health.StateHealthy, string(ownership.Owner))
		}
	}

	auditRepo := database.NewAuditRepo(db)
	notifRepo := database.NewNotificationRepo(db)
	_ = notifRepo
	mgr := firewall.NewManager(backend,
		func(ev firewall.AuditEvent) {
			auditRepo.Insert(database.AuditEntry{
				Actor: ev.Actor, Action: ev.Action, Target: ev.Target,
				Success: ev.Success, Detail: ev.Detail,
			})
		},
		func(ev firewall.IssueEvent) {
			emitStartupIssue(db, ev.Code, ev.Severity, ev.Component, ev.Summary, ev.Remediation)
		})
	mgr.SetOwnership(ownership)

	// HTTP server.
	srv := api.New(cfg, log, db, mgr, ht, diagnostics.NewRegistry())
	httpLn, err := bindListener(cfg)
	if err != nil {
		ht.Set("web", health.StateBlocked, err.Error())
		return err
	}
	defer httpLn.Close()
	ht.Set("web", health.StateHealthy, httpLn.Addr().String())

	// Overall health propagation.
	overall, _ := ht.Snapshot()
	log.Info().Str("event_code", "HEALTH").Str("state", string(overall)).Msg("startup complete")

	errCh := make(chan error, 1)
	httpSrv := &httpServer{ln: httpLn, handler: srv.Router()}
	go func() { errCh <- httpSrv.serve() }()

	<-ctx.Done()
	ht.Set("server", health.StateStopping, "")
	log.Info().Str("event_code", "STOPPED").Msg("fyrwall stopped")
	return nil
}

func bindListener(cfg *config.Config) (netListener, error) {
	return listen(cfg.Server.Bind, cfg.Server.Port, cfg.TLS)
}

func serveRecoveryUI(cfg *config.Config, log zerolog.Logger, ht *health.Tracker, dbErr error) {
	log.Error().Str("event_code", "DB_MIGRATION_FAILED").Err(dbErr).
		Msg("serving recovery page; normal write mode disabled")
	ln, err := bindListener(cfg)
	if err != nil {
		return
	}
	defer ln.Close()
	mux := recoveryHandler(dbErr, ht)
	srv := &httpServer{ln: ln, handler: mux}
	srv.serve()
}

// RunAgent runs the unprivileged agent with its Unix socket. The agent
// verifies its own binary hash on boot before touching the firewall.
func RunAgent(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	log := buildLogger(cfg)
	if err := selfIntegrityCheck(filepathDir(configPath)); err != nil {
		log.Error().Str("event_code", "INTEGRITY_FAILED").Err(err).
			Msg("agent binary tamper check failed; agent refuses to start")
		return fmt.Errorf("agent integrity check failed: %w", err)
	}
	backend, ownership, err := detectBackend(ctx, cfg)
	if err != nil {
		return err
	}
	mgr := firewall.NewManager(backend, nil, nil)
	mgr.SetOwnership(ownership)

	srv := agent.NewServer(mgr, log)
	if err := srv.Listen(""); err != nil {
		return err
	}
	log.Info().Str("event_code", "AGENT_READY").Msg("agent listening")
	return srv.Serve(ctx)
}

// PrintStatus prints a status summary to stdout.
func PrintStatus(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	backend, ownership, err := detectBackend(ctx, cfg)
	if err != nil {
		fmt.Printf("firewall backend: unavailable (%v)\n", err)
		return nil
	}
	st, err := backend.Status(ctx)
	if err != nil {
		fmt.Printf("firewall status: unavailable (%v)\n", err)
	} else {
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
	}
	fmt.Printf("owner: %s writes_allowed: %v\n", ownership.Owner, ownership.WritesAllowed)
	return nil
}

// RunDoctor runs diagnostics read-only with visual output.
func RunDoctor(ctx context.Context) error {
	results := diagnostics.NewRegistry().Run(ctx)
	ui.Banner(os.Stdout, "FYRwall doctor - read-only diagnostics")
	failed := 0
	for _, r := range results {
		ui.PrintStatusLine(os.Stdout, r.Status, r.Code, r.Message)
		if r.Status == "FAIL" {
			failed++
		}
	}
	plat := system.DetectPlatform()
	fmt.Println()
	ui.KV(os.Stdout, "arch/kernel", plat.Arch+"/"+plat.KernelVersion)
	ui.KV(os.Stdout, "init", plat.InitSystem)
	ui.KV(os.Stdout, "environment", mapPlatformEnv(plat))
	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	fmt.Println("\nAll checks passed.")
	return nil
}

func mapPlatformEnv(plat system.Platform) string {
	if plat.IsContainer {
		return "container"
	}
	if plat.IsVM {
		return "VM"
	}
	return "bare metal"
}

// RunPreflight runs diagnostics plus platform checks.
func RunPreflight(ctx context.Context, configPath string) error {
	if err := RunDoctor(ctx); err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if _, _, err := detectBackend(ctx, cfg); err != nil {
		return fmt.Errorf("firewall backend: %w", err)
	}
	fmt.Println("preflight OK")
	return nil
}

// ValidateConfig loads and validates config only.
func ValidateConfig(configPath string) error {
	if _, err := config.Load(configPath); err != nil {
		return err
	}
	fmt.Println("config OK")
	return nil
}

// RunMigrations applies DB migrations.
func RunMigrations(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	fmt.Println("migrations applied")
	return nil
}

// CreateAdmin bootstraps the first admin from an env-var password
// (spec section 29: password never on argv).
func CreateAdmin(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return createAdminIn(db)
}

// RestoreList prints restore points.
func RestoreList(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return restoreList(db)
}

// RestoreCreate makes a manual restore point via the backend.
func RestoreCreate(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	backend, _, err := detectBackend(ctx, cfg)
	if err != nil {
		return err
	}
	snap, err := backend.Snapshot(ctx, "manual CLI")
	if err != nil {
		return err
	}
	fmt.Printf("created restore point %s (hash %s)\n", snap.ID, snap.StateHash[:12])
	return nil
}

// ServiceStatus prints the detected init system and FYRwall service state.
func ServiceStatus(ctx context.Context) error {
	plat := system.DetectPlatform()
	fmt.Printf("init system: %s\n", plat.InitSystem)
	fmt.Printf("container: %v vm: %v\n", plat.IsContainer, plat.IsVM)
	return nil
}
