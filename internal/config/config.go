// Package config loads /etc/fyrwall/config.yaml with env-var overrides
// (FYRWALL_*) and strict validation (spec sections 28 and 94).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PotenFYR-Studios/FYRwall/internal/secureconfig"
	"gopkg.in/yaml.v3"
)

// EnvPrefix is the environment variable namespace.
const EnvPrefix = "FYRWALL_"

// ConfigPath records where the config was loaded from so the GUI-only
// write path can persist to the same file.
var loadedPath string

// ConfigPath returns the path the active config was loaded from.
func ConfigPath() string {
	if loadedPath != "" {
		return loadedPath
	}
	return "/etc/fyrwall/config.yaml"
}

// Config is the full application configuration.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	TLS           TLSConfig           `yaml:"tls"`
	Database      DatabaseConfig      `yaml:"database"`
	Firewall      FirewallConfig      `yaml:"firewall"`
	Service       ServiceConfig       `yaml:"service"`
	Logging       LoggingConfig       `yaml:"logging"`
	RestorePoints RestorePointsConfig `yaml:"restore_points"`
	Security      SecurityConfig      `yaml:"security"`
	Updates       UpdatesConfig       `yaml:"updates"`
}

type ServerConfig struct {
	Bind           string   `yaml:"bind"`
	Port           int      `yaml:"port"`
	PublicURL      string   `yaml:"public_url"`
	TrustedProxies []string `yaml:"trusted_proxies"`
	// Domain binding for public hosting: when set, the server answers only
	// on this Host header (blocks IP-scanning and host-header confusion)
	// and the GUI shows the domain-based URLs.
	Domain string `yaml:"domain"`
	// AgentListener: separate inbound listener for internet-facing agents.
	AgentListener struct {
		Enabled bool   `yaml:"enabled"`
		Bind    string `yaml:"bind"`
		Port    int    `yaml:"port"`
		Domain  string `yaml:"domain"`
	} `yaml:"agent_listener"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type DatabaseConfig struct {
	Driver      string `yaml:"driver"` // sqlite | postgres
	SQLitePath  string `yaml:"sqlite_path"`
	PostgresDSN string `yaml:"postgres_dsn"`
}

type FirewallConfig struct {
	Backend                     string `yaml:"backend"` // auto|ufw|iptables
	SafeApplyTimeoutSeconds     int    `yaml:"safe_apply_timeout_seconds"`
	AllowWriteOnManagerConflict bool   `yaml:"allow_write_on_manager_conflict"`
}

type ServiceConfig struct {
	Autostart bool `yaml:"autostart"`
}

// UpdatesConfig controls the optional new-version notification. Disabled
// by default per the no-telemetry policy (spec section 93).
type UpdatesConfig struct {
	CheckEnabled       bool   `yaml:"check_enabled"`
	ManifestURL        string `yaml:"manifest_url"`
	CheckIntervalHours int    `yaml:"check_interval_hours"`
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"` // json|console
	FileEnabled bool   `yaml:"file_enabled"`
	FilePath    string `yaml:"file_path"`
}

type RestorePointsConfig struct {
	AutoEnabled   bool `yaml:"auto_enabled"`
	RetainAuto    int  `yaml:"retain_automatic"`
	RetainManual  int  `yaml:"retain_manual"`
	RetentionDays int  `yaml:"retention_days"`
}

type SecurityConfig struct {
	SessionIdleTimeoutMinutes int `yaml:"session_idle_timeout_minutes"`
	LoginRateLimitPerMinute   int `yaml:"login_rate_limit_per_minute"`
}

// Default returns the spec-default configuration (loopback bind).
func Default() *Config {
	return &Config{
		Server: ServerConfig{Bind: "127.0.0.1", Port: 7443},
		TLS:    TLSConfig{Enabled: false},
		Database: DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: "/var/lib/fyrwall/fyrwall.db",
		},
		Firewall: FirewallConfig{
			Backend:                     "auto",
			SafeApplyTimeoutSeconds:     60,
			AllowWriteOnManagerConflict: false,
		},
		Service: ServiceConfig{Autostart: true},
		Logging: LoggingConfig{
			Level: "info", Format: "json", FileEnabled: true,
			FilePath: "/var/log/fyrwall/fyrwall.log",
		},
		RestorePoints: RestorePointsConfig{
			AutoEnabled: true, RetainAuto: 50, RetainManual: 20, RetentionDays: 365,
		},
		Security: SecurityConfig{
			SessionIdleTimeoutMinutes: 30,
			LoginRateLimitPerMinute:   5,
		},
		Updates: UpdatesConfig{
			CheckEnabled:       false,
			ManifestURL:        "https://potenfyr-studios.github.io/FYRwall/updates.json",
			CheckIntervalHours: 24,
		},
	}
}

// Load reads configPath, applies env overrides, validates.
// A missing file is not an error: defaults are used.
// Encrypted configs (FYRCFG1 format, managed by secureconfig) are
// decrypted transparently; plaintext configs are migrated to encrypted
// on first load.
func Load(configPath string) (*Config, error) {
	if configPath != "" {
		loadedPath = configPath
	} else {
		configPath = ConfigPath()
	}
	cfg := Default()
	if configPath != "" {
		b, err := readConfigFile(configPath)
		if err == nil {
			if len(b) > 0 {
				if err := yaml.Unmarshal(b, cfg); err != nil {
					return nil, fmt.Errorf("config %s: %w", configPath, err)
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("config %s: %w", configPath, err)
		}
	}
	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyEnv implements FYRWALL_* environment overrides on the YAML surface.
func (c *Config) applyEnv() {
	if v := os.Getenv("FYRWALL_SERVER_BIND"); v != "" {
		c.Server.Bind = v
	}
	if v := os.Getenv("FYRWALL_SERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Server.Port = n
		}
	}
	if v := os.Getenv("FYRWALL_DB_DRIVER"); v != "" {
		c.Database.Driver = v
	}
	if v := os.Getenv("FYRWALL_DB_SQLITE_PATH"); v != "" {
		c.Database.SQLitePath = v
	}
	if v := os.Getenv("FYRWALL_DB_POSTGRES_DSN"); v != "" {
		c.Database.PostgresDSN = v
	}
	if v := os.Getenv("FYRWALL_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("FYRWALL_LOG_FORMAT"); v != "" {
		c.Logging.Format = v
	}
	if v := os.Getenv("FYRWALL_FIREWALL_BACKEND"); v != "" {
		c.Firewall.Backend = v
	}
	if v := os.Getenv("FYRWALL_TLS_ENABLED"); v != "" {
		c.TLS.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("FYRWALL_TLS_CERT"); v != "" {
		c.TLS.CertFile = v
	}
	if v := os.Getenv("FYRWALL_TLS_KEY"); v != "" {
		c.TLS.KeyFile = v
	}
	if v := os.Getenv("FYRWALL_SAFE_APPLY_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.Firewall.SafeApplyTimeoutSeconds = n
		}
	}
	if v := os.Getenv("FYRWALL_UPDATES_CHECK"); v != "" {
		c.Updates.CheckEnabled = v == "true" || v == "1"
	}
	if v := os.Getenv("FYRWALL_UPDATES_MANIFEST_URL"); v != "" {
		c.Updates.ManifestURL = v
	}
	if v := os.Getenv("FYRWALL_DOMAIN"); v != "" {
		c.Server.Domain = v
	}
}

// Validate enforces spec section 28rules and fails fast on unsafe/malformed values.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port out of range: %d", c.Server.Port)
	}
	if c.Server.Bind == "" {
		return fmt.Errorf("server.bind must not be empty")
	}
	if c.Server.Domain != "" {
		if len(c.Server.Domain) > 253 || strings.Contains(c.Server.Domain, "..") {
			return fmt.Errorf("server.domain invalid: %q", c.Server.Domain)
		}
	}
	if c.Server.AgentListener.Enabled {
		if c.Server.AgentListener.Port < 1 || c.Server.AgentListener.Port > 65535 {
			return fmt.Errorf("server.agent_listener.port out of range")
		}
	}
	if !strings.EqualFold(c.Server.Bind, "localhost") && !strings.HasPrefix(c.Server.Bind, "127.") && c.Server.Bind != "::1" {
		if !c.TLS.Enabled {
			// Allowed but must be a conscious choice; install path sets TLS
			// or the operator overrides with an explicit env flag.
			if os.Getenv("FYRWALL_ALLOW_INSECURE_BIND") != "true" {
				return fmt.Errorf("refusing non-loopback bind without TLS (set tls.enabled or FYRWALL_ALLOW_INSECURE_BIND=true)")
			}
		}
	}
	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("tls.enabled requires cert_file and key_file")
		}
		if _, err := os.Stat(c.TLS.CertFile); err != nil {
			return fmt.Errorf("tls.cert_file: %w", err)
		}
		if _, err := os.Stat(c.TLS.KeyFile); err != nil {
			return fmt.Errorf("tls.key_file: %w", err)
		}
	}
	switch strings.ToLower(c.Database.Driver) {
	case "sqlite":
		if c.Database.SQLitePath == "" {
			return fmt.Errorf("database.sqlite_path must not be empty for sqlite driver")
		}
	case "postgres":
		if c.Database.PostgresDSN == "" {
			return fmt.Errorf("database.postgres_dsn must not be empty for postgres driver")
		}
	default:
		return fmt.Errorf("database.driver must be sqlite or postgres, got %q", c.Database.Driver)
	}
	switch strings.ToLower(c.Firewall.Backend) {
	case "auto", "ufw", "iptables":
	default:
		return fmt.Errorf("firewall.backend must be auto|ufw|iptables, got %q", c.Firewall.Backend)
	}
	switch strings.ToLower(c.Logging.Level) {
	case "debug", "info", "warn", "error", "fatal":
	default:
		return fmt.Errorf("logging.level invalid: %q", c.Logging.Level)
	}
	switch strings.ToLower(c.Logging.Format) {
	case "json", "console":
	default:
		return fmt.Errorf("logging.format must be json or console")
	}
	if c.RestorePoints.RetainAuto < 1 {
		return fmt.Errorf("restore_points.retain_automatic must be >= 1")
	}
	if c.RestorePoints.RetainManual < 1 {
		return fmt.Errorf("restore_points.retain_manual must be >= 1")
	}
	if c.Firewall.SafeApplyTimeoutSeconds < 5 || c.Firewall.SafeApplyTimeoutSeconds > 600 {
		return fmt.Errorf("firewall.safe_apply_timeout_seconds must be 5..600")
	}
	if c.Security.LoginRateLimitPerMinute < 1 {
		return fmt.Errorf("security.login_rate_limit_per_minute must be >= 1")
	}
	if c.Updates.CheckEnabled {
		if c.Updates.ManifestURL == "" {
			return fmt.Errorf("updates.manifest_url must not be empty when check_enabled is true")
		}
		if c.Updates.CheckIntervalHours < 1 {
			return fmt.Errorf("updates.check_interval_hours must be >= 1")
		}
	}
	if c.Security.SessionIdleTimeoutMinutes < 1 {
		return fmt.Errorf("security.session_idle_timeout_minutes must be >= 1")
	}
	// SQLite parent dir must be writable by the service account.
	if strings.EqualFold(c.Database.Driver, "sqlite") {
		dir := filepath.Dir(c.Database.SQLitePath)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			if err := checkWritable(dir); err != nil {
				return fmt.Errorf("sqlite dir %s not writable: %w", dir, err)
			}
		}
		// else: dir missing is reported by db init, not config validation.
	}
	return nil
}

// readConfigFile reads config bytes, transparently decrypting an
// encrypted config via secureconfig (which also migrates legacy
// plaintext configs on first load).
func readConfigFile(configPath string) ([]byte, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if len(raw) > 7 && string(raw[:7]) == "FYRCFG1" {
		sc, err := secureconfig.New(configPath)
		if err != nil {
			return nil, err
		}
		return sc.Load()
	}
	// Plaintext: migrate to encrypted if the directory is writable;
	// read-only fallback (containers, non-root CLI) just uses plaintext.
	sc, err := secureconfig.New(configPath)
	if err != nil {
		return raw, nil
	}
	return sc.Load()
}

func checkWritable(dir string) error {
	tmp := filepath.Join(dir, ".fyrwall-write-probe")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(tmp)
	return nil
}
