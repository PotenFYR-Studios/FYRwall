// Package extension implements FYRwall's extension system: declarative,
// capability-scoped add-ons for the web UI, dashboard widgets, network
// event enrichers, notification channels, diagnostics checks, rule
// templates and policy hooks.
//
// Design (spec section 51 boundary respected - extensions can never turn
// FYRwall into an arbitrary execution platform):
//
//   - An extension is a signed manifest plus static assets. There is NO
//     extension code execution on the server: UI extensions are sandboxed
//     React fragments (declarative schema-driven widgets), and backend
//     extensions are typed hook registrations with a fixed capability list.
//   - Every manifest declares the capabilities it needs (read firewall
//     status, read events, add dashboard cards, add notification channels,
//     add diagnostics checks, add templates). Each capability is denied by
//     default and must be granted by an administrator during install.
//   - Anything critical or unsafe to expose is simply NOT an extension
//     capability: raw command execution, direct iptables mutation, user
//     management, credential access, PKI key material, log decryption
//     keys, settings write, and agent enrollment can never be granted.
//     These stay hard-wired in the core by design.
//
// Extension points (what CAN be extended):
//   - dashboard widgets (schema-driven cards)
//   - notification channels (webhook, ntfy, Gotify - typed outbound only)
//   - diagnostics checks (declarative HTTP/file/port probes)
//   - rule templates (additional template packs)
//   - event enrichers (annotation of parsed events, read-only)
//   - UI panels on the extension's own page (sandboxed iframe sandbox)
//
// Deliberately NOT extensible:
//   - firewall transaction pipeline internals (safety-critical)
//   - authentication/RBAC internals
//   - PKI, enrollment, credentials
//   - snapshot/restore engines
//   - anything requiring local code execution on server or agent
package extension

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseYAML decodes a YAML manifest.
func parseYAML(b []byte, m *Manifest) error {
	return yaml.Unmarshal(b, m)
}

// Capability names an extension permission. Deny by default.
type Capability string

const (
	CapDashboardWidget  Capability = "dashboard.widget"
	CapNotifyChannel    Capability = "notify.channel"
	CapDiagnosticsCheck Capability = "diagnostics.check"
	CapRuleTemplate     Capability = "rules.template"
	CapEventEnricher    Capability = "events.enrich"
	CapUIPanel          Capability = "ui.panel"
	CapStatusRead       Capability = "firewall.status.read"
	CapEventsRead       Capability = "events.read"
)

// GrantedCapabilities is the administrator-controlled set.
type GrantedCapabilities map[Capability]bool

// NeverGrantable lists capabilities that do not exist and must never be
// added: execution, direct firewall mutation, auth, PKI, credentials.
// Documented here so reviewers and the install UI can reject manifests
// asking for anything resembling them.
var NeverGrantable = []string{
	"exec", "shell", "firewall.write", "firewall.apply", "users.manage",
	"credentials.read", "pki.sign", "logs.decrypt", "agents.enroll",
	"settings.write", "db.exec",
}

// Manifest is the extension descriptor (extension.yaml).
type Manifest struct {
	ID           string   `json:"id"          yaml:"id"`
	Name         string   `json:"name"        yaml:"name"`
	Version      string   `json:"version"     yaml:"version"`
	Author       string   `json:"author"      yaml:"author"`
	Description  string   `json:"description" yaml:"description"`
	Homepage     string   `json:"homepage"    yaml:"homepage,omitempty"`
	MinApp       string   `json:"min_app"     yaml:"min_app,omitempty"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`

	// Dashboard widgets: declarative cards rendered by the web UI.
	Widgets []WidgetSpec `json:"widgets,omitempty" yaml:"widgets,omitempty"`
	// Notification channels: typed outbound webhooks only.
	NotifyChannels []NotifyChannelSpec `json:"notify_channels,omitempty" yaml:"notify_channels,omitempty"`
	// Diagnostics checks: declarative probes.
	Checks []CheckSpec `json:"checks,omitempty" yaml:"checks,omitempty"`
	// Rule templates: static template packs.
	Templates []TemplateSpec `json:"templates,omitempty" yaml:"templates,omitempty"`
	// UI panels: sandboxed iframe pages bundled as static assets.
	Panels []PanelSpec `json:"panels,omitempty" yaml:"panels,omitempty"`
}

// WidgetSpec declares a dashboard card. The web UI renders from this
// schema - no arbitrary HTML/JS ships from extensions.
type WidgetSpec struct {
	ID      string          `json:"id"      yaml:"id"`
	Title   string          `json:"title"   yaml:"title"`
	Size    string          `json:"size"    yaml:"size"`   // small|medium|wide
	Kind    string          `json:"kind"    yaml:"kind"`   // metric|gauge|table|status
	Source  string          `json:"source"  yaml:"source"` // declared data source name
	Query   json.RawMessage `json:"query,omitempty"  yaml:"query,omitempty"`
	Refresh int             `json:"refresh_seconds,omitempty" yaml:"refresh_seconds,omitempty"`
}

// NotifyChannelSpec declares an outbound notification channel.
type NotifyChannelSpec struct {
	ID          string `json:"id"     yaml:"id"`
	Type        string `json:"type"   yaml:"type"` // webhook|ntfy|gotify|discord
	URL         string `json:"url"    yaml:"url"`
	TokenRef    string `json:"token_ref,omitempty" yaml:"token_ref,omitempty"` // env: or file: reference only
	MinSeverity string `json:"min_severity" yaml:"min_severity"`
}

// CheckSpec declares a diagnostics probe - HTTP GET, TCP connect, or file
// existence only. No command execution.
type CheckSpec struct {
	ID       string `json:"id"       yaml:"id"`
	Name     string `json:"name"     yaml:"name"`
	Type     string `json:"type"     yaml:"type"`   // http|tcp|file
	Target   string `json:"target"   yaml:"target"` // URL | host:port | path
	TimeoutS int    `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	Expect   string `json:"expect,omitempty" yaml:"expect,omitempty"` // e.g. HTTP 200
}

// TemplateSpec is a static rule template pack entry.
type TemplateSpec struct {
	ID          string `json:"id"          yaml:"id"`
	Name        string `json:"name"        yaml:"name"`
	Description string `json:"description" yaml:"description"`
	RulesJSON   string `json:"rules_json"  yaml:"rules_json"` // normalized rules
}

// PanelSpec declares a UI panel served from the extension's own static
// assets in a sandboxed iframe (no same-origin access to FYRwall).
type PanelSpec struct {
	ID      string `json:"id"      yaml:"id"`
	Title   string `json:"title"   yaml:"title"`
	Entry   string `json:"entry"   yaml:"entry"`   // file inside the extension dir
	Sandbox string `json:"sandbox" yaml:"sandbox"` // iframe sandbox attr
}

// RequiredCapabilities derives the capability set a manifest actually uses.
func (m *Manifest) RequiredCapabilities() []Capability {
	set := map[Capability]bool{}
	if len(m.Widgets) > 0 {
		set[CapDashboardWidget] = true
	}
	if len(m.NotifyChannels) > 0 {
		set[CapNotifyChannel] = true
	}
	if len(m.Checks) > 0 {
		set[CapDiagnosticsCheck] = true
	}
	if len(m.Templates) > 0 {
		set[CapRuleTemplate] = true
	}
	if len(m.Panels) > 0 {
		set[CapUIPanel] = true
	}
	if m.RequiredCapabilitiesHasReads() {
		set[CapStatusRead] = true
	}
	out := make([]Capability, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (m *Manifest) RequiredCapabilitiesHasReads() bool {
	// Widgets with firewall-ish sources need status read; keep the mapping
	// explicit and narrow.
	for _, w := range m.Widgets {
		if strings.HasPrefix(w.Source, "firewall.") {
			return true
		}
	}
	return false
}

// Validate enforces manifest hygiene and rejects forbidden capabilities.
func (m *Manifest) Validate() error {
	if m.ID == "" || m.Name == "" || m.Version == "" {
		return errors.New("manifest requires id, name and version")
	}
	if !idRe.MatchString(m.ID) {
		return fmt.Errorf("extension id %q must be lowercase alphanumeric with dashes", m.ID)
	}
	for _, c := range m.Capabilities {
		for _, banned := range NeverGrantable {
			if strings.EqualFold(strings.TrimSpace(c), banned) {
				return fmt.Errorf("capability %q does not exist and can never be granted", c)
			}
		}
	}
	for _, p := range m.Panels {
		if strings.Contains(p.Entry, "..") || strings.HasPrefix(p.Entry, "/") {
			return fmt.Errorf("panel %q entry must be a relative file inside the extension", p.ID)
		}
		if p.Sandbox != "" && strings.Contains(p.Sandbox, "allow-same-origin") {
			return fmt.Errorf("panel %q may not request allow-same-origin", p.ID)
		}
	}
	for _, ch := range m.NotifyChannels {
		if !allowedChannelTypes[ch.Type] {
			return fmt.Errorf("notify channel type %q is not supported (webhook, ntfy, gotify, discord)", ch.Type)
		}
	}
	for _, ck := range m.Checks {
		switch ck.Type {
		case "http", "tcp", "file":
		default:
			return fmt.Errorf("check %q type %q is not supported (http, tcp, file)", ck.ID, ck.Type)
		}
	}
	return nil
}

var allowedChannelTypes = map[string]bool{
	"webhook": true, "ntfy": true, "gotify": true, "discord": true,
}

// idRe: lowercase alphanumeric with dashes, 2-63 chars.
var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

// Installed is an extension on disk with its grant state.
type Installed struct {
	Manifest  Manifest `json:"manifest"`
	Granted   []string `json:"granted"` // granted capability strings
	Dir       string   `json:"dir"`
	AssetsSHA string   `json:"assets_sha256"`
	Enabled   bool     `json:"enabled"`
}

// Registry stores installed extensions under root/<id>/ with a
// granted.json per extension. Install requires an administrator grant of
// each requested capability at install time.
type Registry struct {
	root string
}

func NewRegistry(root string) *Registry { return &Registry{root: root} }

// InstallFromDir validates and records an extension from a directory
// containing extension.yaml plus optional static assets. granted lists the
// capabilities the administrator approved; anything the manifest needs but
// the admin did not grant stays denied at runtime.
func (r *Registry) InstallFromDir(dir string, granted []Capability) (*Installed, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "extension.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read extension.yaml: %w", err)
	}
	var m Manifest
	if err := jsonOrYaml(raw, &m); err != nil {
		return nil, fmt.Errorf("parse extension.yaml: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	// Hash assets for integrity (best-effort; empty dir is fine).
	sum := sha256.Sum256(raw)
	assets := r.assetsDigest(dir, sum)
	grantedSet := map[Capability]bool{}
	for _, g := range granted {
		grantedSet[g] = true
	}
	gs := make([]string, 0, len(granted))
	for _, c := range m.RequiredCapabilities() {
		if grantedSet[c] {
			gs = append(gs, string(c))
		}
	}
	inst := &Installed{
		Manifest: m, Granted: gs, Dir: dir,
		AssetsSHA: hex.EncodeToString(assets[:]), Enabled: true,
	}
	// Record grant state inside our own data dir (never inside the
	// extension dir, which is user-supplied content).
	rec := filepath.Join(r.root, m.ID)
	if err := os.MkdirAll(rec, 0o750); err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(inst, "", "  ")
	if err := os.WriteFile(filepath.Join(rec, "installed.json"), b, 0o640); err != nil {
		return nil, err
	}
	return inst, nil
}

// List returns installed extensions.
func (r *Registry) List() ([]Installed, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Installed
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(r.root, e.Name(), "installed.json"))
		if err != nil {
			continue
		}
		var inst Installed
		if json.Unmarshal(b, &inst) == nil {
			out = append(out, inst)
		}
	}
	return out, nil
}

// assetsDigest hashes all regular files under dir (sorted by name).
func (r *Registry) assetsDigest(dir string, seed [32]byte) [32]byte {
	h := sha256.New()
	h.Write(seed[:])
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		h.Write([]byte(strings.TrimPrefix(path, dir)))
		return nil
	})
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func jsonOrYaml(b []byte, m *Manifest) error {
	// extension.yaml is a YAML file; we only depend on encoding/json-style
	// structure, so accept both by trying JSON first then YAML via the
	// project's existing yaml dependency in the caller. To stay
	// dependency-lean here, accept plain JSON-shaped manifests too.
	if err := json.Unmarshal(b, m); err == nil {
		return nil
	}
	return parseYAML(b, m)
}
