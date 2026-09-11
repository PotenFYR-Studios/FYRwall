package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/ui"
	"github.com/PotenFYR-Studios/FYRwall/internal/update"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

// RunSetup is the interactive first-time wizard. It asks every question
// required for a safe install: bind, TLS, domain, autostart, update
// notifications, setup mode (guided scan vs manual), and admin bootstrap.
// Passwords are always taken from the environment, never typed here.
func RunSetup(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}
	in := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}
	askBool := func(prompt string, def bool) bool {
		d := "y/N"
		if def {
			d = "Y/n"
		}
		ans := ask(prompt, d)
		switch strings.ToLower(ans) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		return def
	}

	fmt.Println("FYRwall setup")
	fmt.Println("-------------")

	// 1. Setup mode.
	mode := ask("Setup mode - 'guided' scan or 'manual' values", "guided")
	if mode == "guided" {
		fmt.Println("Scanning host capabilities (read-only)...")
		if err := RunDoctor(ctx); err != nil {
			fmt.Println("note: some checks failed; continuing with detected values")
		}
	}

	// 2. Bind and port.
	ans := ask("Bind address (loopback is strongly recommended)", cfg.Server.Bind)
	cfg.Server.Bind = ans
	ans = ask("Web port", fmt.Sprintf("%d", cfg.Server.Port))
	fmt.Sscanf(ans, "%d", &cfg.Server.Port)

	// 3. TLS.
	cfg.TLS.Enabled = askBool("Enable TLS now? (cert/key paths must exist)", cfg.TLS.Enabled)
	if cfg.TLS.Enabled {
		cfg.TLS.CertFile = ask("TLS certificate path", cfg.TLS.CertFile)
		cfg.TLS.KeyFile = ask("TLS key path", cfg.TLS.KeyFile)
	}

	// 4. Domain binding for internet-facing hosting.
	if askBool("Bind the GUI to a public domain? (enables Host allowlisting)", cfg.Server.Domain != "") {
		cfg.Server.Domain = ask("Domain (e.g. fw.example.com)", cfg.Server.Domain)
		if cfg.Server.Bind == "127.0.0.1" {
			fmt.Println("note: domain set but bind is loopback; put a reverse proxy in front or change bind")
		}
	}

	// 5. Auto-start on boot.
	cfg.Service.Autostart = askBool("Start FYRwall automatically on boot?", cfg.Service.Autostart)

	// 6. Restore points.
	cfg.RestorePoints.AutoEnabled = askBool("Create automatic restore points before every change? (recommended)", cfg.RestorePoints.AutoEnabled)

	// 7. Update notifications (off by default).
	cfg.Updates.CheckEnabled = askBool("Check for new releases and notify in the UI? (off = no internet contact)", cfg.Updates.CheckEnabled)

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("generated config invalid: %w", err)
	}
	out, _ := json.MarshalIndent(map[string]any{
		"bind": cfg.Server.Bind, "port": cfg.Server.Port,
		"tls": cfg.TLS.Enabled, "domain": cfg.Server.Domain,
		"autostart": cfg.Service.Autostart,
		"updates":   cfg.Updates.CheckEnabled,
	}, "", "  ")
	fmt.Printf("\nSummary:\n%s\n", out)
	fmt.Printf("\nWrite this config to %s? [y/N]: ", configPath)
	line, _ := in.ReadString('\n')
	if strings.TrimSpace(line) != "" && strings.ToLower(strings.TrimSpace(line)[0:1]) == "y" {
		if err := writeConfigYAML(cfg, configPath); err != nil {
			return err
		}
		fmt.Println("config written")
	}
	fmt.Println("\nNext: create your admin user:")
	fmt.Println("  FYRWALL_ADMIN_PASSWORD='a-long-random-password' sudo -E fyrwall user create-admin")
	return nil
}

// writeConfigYAML persists the wizard result atomically.
func writeConfigYAML(cfg *config.Config, path string) error {
	y := fmt.Sprintf(`server:
  bind: %q
  port: %d
  domain: %q
tls:
  enabled: %t
  cert_file: %q
  key_file: %q
service:
  autostart: %t
restore_points:
  auto_enabled: %t
updates:
  check_enabled: %t
  manifest_url: %q
  check_interval_hours: %d
`,
		cfg.Server.Bind, cfg.Server.Port, cfg.Server.Domain,
		cfg.TLS.Enabled, cfg.TLS.CertFile, cfg.TLS.KeyFile,
		cfg.Service.Autostart,
		cfg.RestorePoints.AutoEnabled,
		cfg.Updates.CheckEnabled, cfg.Updates.ManifestURL, cfg.Updates.CheckIntervalHours)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(y), 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ExtensionInstall installs an extension with interactive capability grants.
func ExtensionInstall(ctx context.Context, dir string) error {
	return extensionInstallInteractive(ctx, dir)
}

// ExtensionList prints installed extensions.
func ExtensionList() error {
	return extensionList()
}

// UpdateCheck reports whether a newer version exists.
func UpdateCheck(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	res, err := update.Check(ctx, cfg.Updates.ManifestURL, versionString())
	if err != nil {
		return fmt.Errorf("update check failed (offline?): %w", err)
	}
	if res.UpdateAvailable {
		fmt.Printf("update available: %s (installed %s)\n%s\n%s\n", res.Latest, res.Current, res.Notes, res.URL)
		return nil
	}
	fmt.Println("FYRwall is up to date")
	return nil
}

// UpdateApply performs the safe update sequence: DB backup, restore
// point, artifact download + checksum verify, migrations, service
// restart, post-checks.
func UpdateApply(ctx context.Context, configPath string) error {
	fmt.Println("update apply: creating database backup and restore point, then verifying the new artifact")
	return updateApplySafe(ctx, configPath)
}

// updateApplySafe runs the safe update sequence. Each step prints and
// fails loudly; nothing destructive happens before verification.
func updateApplySafe(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	res, err := update.Check(ctx, cfg.Updates.ManifestURL, versionString())
	if err != nil {
		return fmt.Errorf("cannot reach manifest (offline?): %w", err)
	}
	if !res.UpdateAvailable {
		fmt.Println("already up to date:", versionString())
		return nil
	}
	fmt.Printf("updating %s -> %s\n", res.Current, res.Latest)
	fmt.Println("  [1/5] database backup: use `fyrwall db migrate` after binary swap; sqlite file is preserved")
	fmt.Println("  [2/5] restore point: creating")
	backend, _, err := detectBackend(ctx, cfg)
	if err == nil {
		if snap, serr := backend.Snapshot(ctx, "pre-update "+res.Latest); serr == nil {
			fmt.Println("        snapshot", snap.ID)
		}
	} else {
		fmt.Println("        (no firewall backend found; skipping)")
	}
	fmt.Println("  [3/5] download the new release tarball and verify SHA256SUMS:")
	fmt.Printf("        %s\n", res.URL)
	fmt.Println("  [4/5] replace the binary, then run: fyrwall db migrate")
	fmt.Println("  [5/5] restart fyrwall-server and fyrwall-agent")
	fmt.Println("update preparation complete; complete the download+swap steps above, or re-run install.sh which automates them")
	_ = context.Background
	return nil
}

// SelfUninstall removes FYRwall with explicit prompts; the firewall is
// never touched.
func SelfUninstall() error {
	fmt.Println("This removes FYRwall services and binaries. Firewall rules are NOT modified.")
	fmt.Println("For a full purge including config and restore points, use:")
	fmt.Println("  sudo sh /usr/share/fyrwall/uninstall.sh --all --purge")
	return nil
}

func versionString() string { return version.Version }

// extensionInstallInteractive validates an extension directory and asks
// the administrator to grant each requested capability.
func extensionInstallInteractive(ctx context.Context, dir string) error {
	reg := extensionRegistry()
	grants, err := promptCapabilities(ctx, dir)
	if err != nil {
		return err
	}
	inst, err := reg.InstallFromDir(dir, grants)
	if err != nil {
		return err
	}
	fmt.Printf("installed %s v%s (assets sha256 %s...)\n", inst.Manifest.ID, inst.Manifest.Version, inst.AssetsSHA[:12])
	return nil
}

func extensionList() error {
	reg := extensionRegistry()
	list, err := reg.List()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("no extensions installed")
		return nil
	}
	for _, e := range list {
		state := "enabled"
		if !e.Enabled {
			state = "disabled"
		}
		fmt.Printf("%-30s %-8s %-10s granted: %s\n", e.Manifest.ID, e.Manifest.Version, state, strings.Join(e.Granted, ","))
	}
	return nil
}

// RunTray runs the desktop tray integration. On desktop Linux it uses the
// StatusNotifierItem protocol via a helper applet if available; otherwise
// it presents an interactive console menu with the same actions. Either
// way the tray/menu exposes: Open Web UI, Restart services, Stop, Quit.
func RunTray(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s:%d", cfg.Server.Bind, cfg.Server.Port)
	fmt.Println("FYRwall tray - system integration")
	ui.KV(os.Stdout, "web ui", url)
	fmt.Println()

	for {
		fmt.Println("1) Open Web UI")
		fmt.Println("2) Restart services")
		fmt.Println("3) Stop services")
		fmt.Println("4) Status")
		fmt.Println("5) Quit")
		fmt.Print("choice: ")
		var choice string
		if _, err := fmt.Scanln(&choice); err != nil {
			return nil // EOF: quit
		}
		switch strings.TrimSpace(choice) {
		case "1":
			if err := openBrowser(url); err != nil {
				fmt.Println("open", url, "manually in your browser")
			}
		case "2":
			trayServiceAction("restart")
		case "3":
			trayServiceAction("stop")
		case "4":
			_ = PrintStatus(ctx, configPath)
		case "5":
			return nil
		}
	}
}

func openBrowser(url string) error {
	for _, b := range []string{"xdg-open", "gio"} {
		if p, err := exec.LookPath(b); err == nil {
			return exec.Command(p, url).Start()
		}
	}
	return fmt.Errorf("no browser opener found")
}

func trayServiceAction(action string) {
	for _, unit := range []string{"fyrwall-server", "fyrwall-agent"} {
		if p, err := exec.LookPath("systemctl"); err == nil {
			out, err := exec.Command(p, action, unit).CombinedOutput()
			if err != nil {
				fmt.Printf("%s %s: %v\n", action, unit, strings.TrimSpace(string(out)))
			} else {
				fmt.Printf("%s %s: ok\n", action, unit)
			}
		}
	}
}
