// Package ufw implements the FirewallBackend for Uncomplicated Firewall.
// All UFW interaction goes through executil with argument arrays; the
// parser treats command output as untrusted input (spec sections 6, 11, 43).
package ufw

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/system/exec"
)

// Backend is the UFW adapter. bin is the resolved ufw binary path; tests
// inject a stub path so no root firewall state is touched in CI.
type Backend struct {
	bin string
}

// New resolves the ufw binary or returns ErrBackendUnavailable.
func New() (*Backend, error) {
	bin, err := exec.LookPath("ufw")
	if err != nil {
		// Spec section 8.2: discover common locations rather than assume.
		for _, p := range []string{"/usr/sbin/ufw", "/usr/bin/ufw", "/sbin/ufw"} {
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return &Backend{bin: p}, nil
			}
		}
		return nil, fmt.Errorf("%w: ufw binary not found", firewall.ErrBackendUnavailable)
	}
	return &Backend{bin: bin}, nil
}

// NewWithBinary is used by tests to inject a stub executable.
func NewWithBinary(path string) *Backend { return &Backend{bin: path} }

func (b *Backend) Name() string { return "ufw" }

// Detect reports whether UFW exists and its version.
func (b *Backend) Detect(ctx context.Context) firewall.DetectionResult {
	res := firewall.DetectionResult{Name: "ufw"}
	res.BinaryPath = b.bin
	if b.bin == "" {
		return res
	}
	out, err := exec.Run(ctx, 10*time.Second, b.bin, "version")
	if err != nil {
		res.Details = err.Error()
		return res
	}
	res.Found = true
	res.Version = firstLine(out.Stdout)
	return res
}

// Status parses `ufw status verbose`.
func (b *Backend) Status(ctx context.Context) (firewall.FirewallStatus, error) {
	out, err := exec.Run(ctx, 15*time.Second, b.bin, "status", "verbose")
	if err != nil {
		return firewall.FirewallStatus{}, fmt.Errorf("%w: %s", firewall.ErrBackendUnavailable, err)
	}
	return parseStatusVerbose(out.Stdout)
}

// ListRules parses `ufw status numbered`.
func (b *Backend) ListRules(ctx context.Context) ([]firewall.Rule, error) {
	out, err := exec.Run(ctx, 15*time.Second, b.bin, "status", "numbered")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", firewall.ErrBackendUnavailable, err)
	}
	return parseStatusNumbered(out.Stdout)
}

// Validate runs static validation plus conflict analysis. UFW itself has no
// dry-run; structural validation happens in the shared engine.
func (b *Backend) Validate(ctx context.Context, tx firewall.Transaction) firewall.ValidationResult {
	for _, action := range tx.Actions {
		if action.Op != "add" && action.Op != "update" && action.Op != "delete" {
			return firewall.ValidationResult{Valid: false, Errors: []string{"UFW supports add, update, and delete transactions"}}
		}
	}
	rules, err := b.ListRules(ctx)
	if err != nil {
		return firewall.ValidationResult{Valid: false, Errors: []string{err.Error()}}
	}
	return firewall.ValidateTx(tx, rules)
}

// Snapshot captures the UFW state plus iptables-save output for rollback.
func (b *Backend) Snapshot(ctx context.Context, reason string) (firewall.Snapshot, error) {
	snap := firewall.Snapshot{
		ID: newSnapshotID(), Reason: reason, Backend: "ufw",
		CreatedAt: time.Now().UTC(),
	}
	out, err := exec.Run(ctx, 15*time.Second, b.bin, "status", "verbose")
	if err == nil {
		snap.UFWStatus = out.Stdout
	}
	if out, err := exec.Run(ctx, 15*time.Second, "iptables-save"); err == nil {
		snap.IptablesSave = out.Stdout
	}
	if out, err := exec.Run(ctx, 15*time.Second, "ip6tables-save"); err == nil {
		snap.IP6TablesSave = out.Stdout
	}
	snap.StateHash = hashSnapshot(snap)
	return snap, nil
}

// Apply executes the transaction against UFW with restore-point protection
// handled by the caller (FirewallManager).
func (b *Backend) Apply(ctx context.Context, tx firewall.Transaction) (firewall.ApplyResult, error) {
	start := time.Now()
	applied := 0
	var warnings []string
	for _, a := range tx.Actions {
		if err := b.applyAction(ctx, a); err != nil {
			return firewall.ApplyResult{TxnID: tx.ID, Applied: applied, Warnings: warnings}, err
		}
		applied++
	}
	out, err := exec.Run(ctx, 15*time.Second, b.bin, "status", "verbose")
	if err != nil {
		return firewall.ApplyResult{TxnID: tx.ID, Applied: applied}, err
	}
	stateHash := hashSnapshot(firewall.Snapshot{UFWStatus: out.Stdout})
	return firewall.ApplyResult{
		TxnID: tx.ID, Applied: applied,
		StateHash: stateHash, DurationMs: time.Since(start).Milliseconds(), Warnings: warnings,
	}, nil
}

func (b *Backend) applyAction(ctx context.Context, a firewall.TxnAction) error {
	switch a.Op {
	case "add", "update":
		r := a.Rule
		if r == nil {
			return fmt.Errorf("add/update action without rule")
		}
		args, err := toUFWArgs(r)
		if err != nil {
			return err
		}
		if a.Op == "update" {
			if a.RuleID == "" {
				return fmt.Errorf("update action requires backend rule ID")
			}
			if out, delErr := exec.Run(ctx, 30*time.Second, b.bin, "delete", a.RuleID); delErr != nil {
				return fmt.Errorf("%w: ufw update delete: %s", firewall.ErrCommandFailed, firstLine(out.Stderr))
			}
		}
		out, err := exec.Run(ctx, 30*time.Second, b.bin, args...)
		if err != nil {
			return fmt.Errorf("%w: ufw %s: %s", firewall.ErrCommandFailed, a.Op, firstLine(out.Stderr))
		}
		return nil
	case "delete":
		if a.RuleID != "" {
			out, err := exec.Run(ctx, 30*time.Second, b.bin, "delete", a.RuleID)
			if err != nil {
				return fmt.Errorf("%w: ufw delete: %s", firewall.ErrCommandFailed, firstLine(out.Stderr))
			}
			return nil
		}
		if a.Rule != nil {
			args, err := toUFWArgs(a.Rule)
			if err != nil {
				return err
			}
			out, err := exec.Run(ctx, 30*time.Second, b.bin, append([]string{"delete"}, args...)...)
			if err != nil {
				return fmt.Errorf("%w: ufw delete: %s", firewall.ErrCommandFailed, firstLine(out.Stderr))
			}
			return nil
		}
		return fmt.Errorf("delete action without rule reference")
	default:
		return fmt.Errorf("unsupported ufw action %q", a.Op)
	}
}

// toUFWSpec converts a normalized rule to the UFW CLI grammar.
func toUFWSpec(r *firewall.Rule) (string, error) {
	args, err := toUFWArgs(r)
	return strings.Join(args, " "), err
}

func toUFWArgs(r *firewall.Rule) ([]string, error) {
	var parts []string
	switch r.Direction {
	case firewall.DirIn:
		switch r.Action {
		case firewall.ActionAllow:
			parts = append(parts, "allow")
		case firewall.ActionReject:
			parts = append(parts, "reject")
		default:
			parts = append(parts, "deny")
		}
	case firewall.DirOut:
		verb := "allow"
		if r.Action == firewall.ActionDeny || r.Action == firewall.ActionDrop {
			verb = "deny"
		}
		parts = append(parts, verb, "out")
	case firewall.DirForward:
		verb := "allow"
		if r.Action == firewall.ActionDeny || r.Action == firewall.ActionDrop {
			verb = "deny"
		}
		parts = append(parts, "route", verb)
	}
	if r.InterfaceIn != "" {
		parts = append(parts, "in", "on", r.InterfaceIn)
	}
	if r.InterfaceOut != "" {
		parts = append(parts, "out", "on", r.InterfaceOut)
	}
	if r.Protocol != firewall.ProtoAny {
		parts = append(parts, "proto", string(r.Protocol))
	}
	if r.Source != "" && r.Source != "any" {
		parts = append(parts, "from", r.Source)
		if r.SourcePort != "" {
			parts = append(parts, "port", r.SourcePort)
		}
	}
	dst := r.Destination
	if dst == "" || dst == "any" {
		dst = "any"
	}
	parts = append(parts, "to", dst)
	if r.DestinationPort != "" {
		parts = append(parts, "port", r.DestinationPort)
	}
	return parts, nil
}

// Verify compares expected state hash against fresh UFW output.
func (b *Backend) Verify(ctx context.Context, expected firewall.StateHash) (firewall.VerifyResult, error) {
	out, err := exec.Run(ctx, 15*time.Second, b.bin, "status", "verbose")
	if err != nil {
		return firewall.VerifyResult{}, err
	}
	snap := firewall.Snapshot{UFWStatus: out.Stdout}
	observed := hashSnapshot(snap)
	return firewall.VerifyResult{
		Match:    observed == string(expected),
		Observed: observed, Expected: string(expected),
	}, nil
}

// Restore replays iptables-save content captured in the snapshot. UFW rule
// replay uses `ufw reset` only in a dedicated recovery path guarded by the
// privileged helper; the normal path restores the raw netfilter state.
func (b *Backend) Restore(ctx context.Context, snap firewall.Snapshot) error {
	if snap.IptablesSave != "" {
		if err := restoreIptables(ctx, "iptables-restore", snap.IptablesSave); err != nil {
			return err
		}
	}
	if snap.IP6TablesSave != "" {
		if err := restoreIptables(ctx, "ip6tables-restore", snap.IP6TablesSave); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) Reload(ctx context.Context) error {
	out, err := exec.Run(ctx, 30*time.Second, b.bin, "reload")
	if err != nil {
		return fmt.Errorf("%w: %s", firewall.ErrCommandFailed, firstLine(out.Stderr))
	}
	return nil
}

func (b *Backend) Restart(ctx context.Context) error {
	if out, err := exec.Run(ctx, 10*time.Second, b.bin, "disable"); err != nil {
		_ = out
		return err
	}
	out, err := exec.Run(ctx, 30*time.Second, b.bin, "enable")
	if err != nil {
		return fmt.Errorf("%w: %s", firewall.ErrCommandFailed, firstLine(out.Stderr))
	}
	return nil
}

func restoreIptables(ctx context.Context, bin, content string) error {
	out, err := exec.RunInput(ctx, 60*time.Second, content, bin)
	if err != nil {
		return fmt.Errorf("%w: %s: %s", firewall.ErrCommandFailed, bin, firstLine(out.Stderr))
	}
	return nil
}

// parseStatusVerbose handles `ufw status verbose` output.
func parseStatusVerbose(out string) (firewall.FirewallStatus, error) {
	st := firewall.FirewallStatus{Backend: "ufw", CheckedAt: time.Now().UTC()}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Status:"):
			st.Enabled = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "Status:")), "active")
		case strings.HasPrefix(line, "Default:"):
			st.DefaultPolicies = parseDefaultLine(line)
		}
	}
	return st, nil
}

// parseDefaultLine handles e.g. "Default: deny (incoming), allow (outgoing), disabled (routed)".
func parseDefaultLine(line string) []firewall.DefaultPolicy {
	var out []firewall.DefaultPolicy
	body := strings.TrimSpace(strings.TrimPrefix(line, "Default:"))
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		action := part
		var dirLabel string
		if i := strings.Index(part, "("); i >= 0 {
			action = strings.TrimSpace(part[:i])
			dirLabel = strings.Trim(part[i:], "()")
		}
		var dir firewall.Direction
		switch {
		case strings.HasPrefix(dirLabel, "incoming"):
			dir = firewall.DirIn
		case strings.HasPrefix(dirLabel, "outgoing"):
			dir = firewall.DirOut
		case strings.HasPrefix(dirLabel, "routed"):
			dir = firewall.DirForward
		default:
			continue
		}
		act := firewall.ActionDeny
		if strings.EqualFold(action, "allow") {
			act = firewall.ActionAllow
		}
		out = append(out, firewall.DefaultPolicy{Direction: dir, Action: act})
	}
	return out
}

// parseStatusNumbered handles `ufw status numbered` table output with
// lines like:  [ 1] 22/tcp                     ALLOW IN    Anywhere
func parseStatusNumbered(out string) ([]firewall.Rule, error) {
	var rules []firewall.Rule
	priority := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		closeIdx := strings.IndexByte(line, ']')
		if closeIdx < 0 {
			continue
		}
		backendNumber, err := strconv.Atoi(strings.TrimSpace(line[1:closeIdx]))
		if err != nil {
			continue // v6 block header "[ N]" after comment marker is still a rule line; non-numeric is a section separator
		}
		rest := strings.Fields(line[closeIdx+1:])
		if len(rest) < 3 {
			continue
		}
		rule := firewall.Rule{
			ID: firewall.NewRuleID(), BackendID: strconv.Itoa(backendNumber), Backend: "ufw",
			Enabled: true, Priority: priority,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		priority++
		rule.Protocol, rule.SourcePort, rule.DestinationPort = splitPortSpec(rest[0])

		actionIdx := -1
		for i, f := range rest[1:] {
			up := strings.ToUpper(f)
			if up == "ALLOW" || up == "DENY" || up == "REJECT" || up == "LIMIT" {
				actionIdx = i + 1
				break
			}
		}
		if actionIdx < 0 || actionIdx+1 >= len(rest) {
			continue
		}
		action := strings.ToUpper(rest[actionIdx])
		switch action {
		case "ALLOW":
			rule.Action = firewall.ActionAllow
		case "DENY":
			rule.Action = firewall.ActionDeny
		case "REJECT":
			rule.Action = firewall.ActionReject
		case "LIMIT":
			rule.Action = firewall.ActionAllow
			rule.Comment = "ufw limit"
		}
		dirTok := strings.ToUpper(rest[actionIdx+1])
		switch {
		case strings.HasPrefix(dirTok, "IN"):
			rule.Direction = firewall.DirIn
		case strings.HasPrefix(dirTok, "OUT"):
			rule.Direction = firewall.DirOut
		case strings.HasPrefix(dirTok, "FWD"), strings.HasPrefix(dirTok, "ROUTE"):
			rule.Direction = firewall.DirForward
		default:
			rule.Direction = firewall.DirIn
		}
		// Remaining tokens: "Anywhere" or "IP", plus "(v6)" marker and
		// a trailing comment in parentheses which may contain spaces.
		trailer := strings.Join(rest[actionIdx+2:], " ")
		for _, tok := range rest[actionIdx+2:] {
			if tok == "(v6)" {
				rule.Family = firewall.FamilyIPv6
			} else if tok != "Anywhere" && !strings.HasPrefix(tok, "(") && rule.Source == "" {
				rule.Source = tok
			}
		}
		if i := strings.IndexByte(trailer, '('); i >= 0 && strings.HasSuffix(trailer, ")") {
			rule.Comment = trailer[i+1 : len(trailer)-1]
		}
		if rule.Family == "" {
			rule.Family = firewall.FamilyIPv4
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// splitPortSpec handles "22/tcp", "any", "137,138/udp", "1000:2000/tcp".
func splitPortSpec(spec string) (firewall.Protocol, string, string) {
	var proto firewall.Protocol = firewall.ProtoAny
	var port string
	if i := strings.IndexByte(spec, '/'); i >= 0 {
		port = spec[:i]
		switch strings.ToLower(spec[i+1:]) {
		case "tcp":
			proto = firewall.ProtoTCP
		case "udp":
			proto = firewall.ProtoUDP
		default:
			proto = firewall.ProtoCustom
		}
	} else {
		port = spec
	}
	if port == "any" || port == "Anywhere" {
		port = ""
	}
	return proto, port, port
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

func newSnapshotID() string {
	return "snap-" + firewall.NewRuleID()
}

// hashSnapshot derives a stable state hash from the captured raw output.
func hashSnapshot(s firewall.Snapshot) string {
	h := sha256.New()
	h.Write([]byte(s.UFWStatus))
	h.Write([]byte(s.IptablesSave))
	h.Write([]byte(s.IP6TablesSave))
	return hex.EncodeToString(h.Sum(nil))
}
