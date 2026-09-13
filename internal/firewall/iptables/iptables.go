// Package iptables implements the FirewallBackend for raw iptables,
// including legacy vs iptables-nft detection and an iptables-save parser.
// Parser output is treated as untrusted input (spec sections 6, 40, 43).
package iptables

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/system/exec"
)

// Mode distinguishes legacy vs nft-backed iptables.
type Mode string

const (
	ModeLegacy  Mode = "legacy"
	ModeNFT     Mode = "nft"
	ModeUnknown Mode = "unknown"
)

// Tools bundles resolved binary paths for the iptables family.
type Tools struct {
	Iptables         string
	IP6Tables        string
	IptablesSave     string
	IptablesRestore  string
	IP6TablesSave    string
	IP6TablesRestore string
	LegacySave       string // iptables-legacy-save if present
	Mode             Mode
	Version          string
}

// Backend is the raw iptables adapter.
type Backend struct {
	tools Tools
}

// DetectTools discovers the iptables binaries and the legacy/nft flavor.
func DetectTools(ctx context.Context) (Tools, error) {
	var t Tools
	found := false
	for _, pair := range []struct {
		name string
		dst  *string
	}{
		{"iptables", &t.Iptables}, {"ip6tables", &t.IP6Tables},
		{"iptables-save", &t.IptablesSave}, {"iptables-restore", &t.IptablesRestore},
		{"ip6tables-save", &t.IP6TablesSave}, {"ip6tables-restore", &t.IP6TablesRestore},
	} {
		if p, err := exec.LookPath(pair.name); err == nil {
			*pair.dst = p
			found = true
		} else {
			// Spec section 8.2 fallback paths.
			for _, alt := range []string{"/usr/sbin/" + pair.name, "/usr/bin/" + pair.name, "/sbin/" + pair.name} {
				if fi, err := os.Stat(alt); err == nil && !fi.IsDir() {
					*pair.dst = alt
					found = true
					break
				}
			}
		}
	}
	if !found {
		return t, fmt.Errorf("%w: no iptables binaries found", firewall.ErrBackendUnavailable)
	}

	// Flavor detection: `iptables --version` reports (legacy) or (nf_tables).
	if t.Iptables != "" {
		out, err := exec.Run(ctx, 10*time.Second, t.Iptables, "--version")
		if err == nil {
			t.Version = strings.TrimSpace(out.Stdout)
			switch {
			case strings.Contains(out.Stdout, "nf_tables"), strings.Contains(out.Stdout, "nft"):
				t.Mode = ModeNFT
			case strings.Contains(out.Stdout, "legacy"):
				t.Mode = ModeLegacy
			default:
				t.Mode = ModeUnknown
			}
		}
	}
	if p, err := exec.LookPath("iptables-legacy-save"); err == nil {
		t.LegacySave = p
	}
	return t, nil
}

// New builds an adapter from detected tools.
func New(tools Tools) *Backend { return &Backend{tools: tools} }

func (b *Backend) Name() string { return "iptables" }

func (b *Backend) Detect(ctx context.Context) firewall.DetectionResult {
	res := firewall.DetectionResult{Name: "iptables", BinaryPath: b.tools.Iptables, Version: b.tools.Version}
	res.Found = b.tools.Iptables != ""
	if b.tools.Mode == ModeNFT {
		res.Details = "iptables-nft compatibility layer"
	} else if b.tools.Mode == ModeLegacy {
		res.Details = "iptables-legacy"
	}
	return res
}

// Status reads current state via iptables-save.
func (b *Backend) Status(ctx context.Context) (firewall.FirewallStatus, error) {
	st := firewall.FirewallStatus{Backend: "iptables", CheckedAt: time.Now().UTC()}
	out, err := exec.Run(ctx, 15*time.Second, b.tools.IptablesSave)
	if err != nil {
		return st, fmt.Errorf("%w: %s", firewall.ErrBackendUnavailable, err)
	}
	rules := ParseSave(out.Stdout)
	st.RuleCount = len(rules)
	st.IPv4Ready = true
	st.RawVersion = b.tools.Version

	// Default policies appear as "-P INPUT ACCEPT" lines.
	for _, line := range strings.Split(out.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-P ") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		dir := chainToDirection(f[1])
		if dir == "" {
			continue
		}
		act := firewall.ActionDeny
		if f[2] == "ACCEPT" {
			act = firewall.ActionAllow
		}
		st.DefaultPolicies = append(st.DefaultPolicies, firewall.DefaultPolicy{Direction: dir, Action: act})
	}
	if b.tools.IP6TablesSave != "" {
		if _, err := exec.Run(ctx, 15*time.Second, b.tools.IP6TablesSave); err == nil {
			st.IPv6Ready = true
		}
	}
	st.Enabled = st.RuleCount > 0 || len(st.DefaultPolicies) > 0
	return st, nil
}

// ListRules returns normalized rules parsed from iptables-save.
func (b *Backend) ListRules(ctx context.Context) ([]firewall.Rule, error) {
	out, err := exec.Run(ctx, 15*time.Second, b.tools.IptablesSave)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", firewall.ErrBackendUnavailable, err)
	}
	rules := ParseSave(out.Stdout)
	// IPv6 rules get their own family tag.
	if b.tools.IP6TablesSave != "" {
		if out6, err := exec.Run(ctx, 15*time.Second, b.tools.IP6TablesSave); err == nil {
			for _, r := range ParseSave6(out6.Stdout) {
				r.Family = firewall.FamilyIPv6
				rules = append(rules, r)
			}
		}
	}
	return rules, nil
}

// Validate delegates to the shared engine.
func (b *Backend) Validate(ctx context.Context, tx firewall.Transaction) firewall.ValidationResult {
	for _, action := range tx.Actions {
		if action.Op != "add" && action.Op != "delete" {
			return firewall.ValidationResult{Valid: false, Errors: []string{"iptables supports add and delete transactions"}}
		}
	}
	rules, err := b.ListRules(ctx)
	if err != nil {
		return firewall.ValidationResult{Valid: false, Errors: []string{err.Error()}}
	}
	return firewall.ValidateTx(tx, rules)
}

// Snapshot captures iptables-save output for both families.
func (b *Backend) Snapshot(ctx context.Context, reason string) (firewall.Snapshot, error) {
	snap := firewall.Snapshot{ID: "snap-" + firewall.NewRuleID(), Reason: reason, Backend: "iptables", CreatedAt: time.Now().UTC()}
	if b.tools.IptablesSave != "" {
		if out, err := exec.Run(ctx, 15*time.Second, b.tools.IptablesSave); err == nil {
			snap.IptablesSave = out.Stdout
		}
	}
	if b.tools.IP6TablesSave != "" {
		if out, err := exec.Run(ctx, 15*time.Second, b.tools.IP6TablesSave); err == nil {
			snap.IP6TablesSave = out.Stdout
		}
	}
	h := sha256.Sum256([]byte(snap.IptablesSave + "\x00" + snap.IP6TablesSave))
	snap.StateHash = hex.EncodeToString(h[:])
	return snap, nil
}

// Apply inserts/deletes iptables rules via argument-array invocations.
func (b *Backend) Apply(ctx context.Context, tx firewall.Transaction) (firewall.ApplyResult, error) {
	start := time.Now()
	applied := 0
	for _, a := range tx.Actions {
		if err := b.applyAction(ctx, a); err != nil {
			return firewall.ApplyResult{TxnID: tx.ID, Applied: applied}, err
		}
		applied++
	}
	out, _ := exec.Run(ctx, 15*time.Second, b.tools.IptablesSave)
	h := sha256.Sum256([]byte(out.Stdout))
	return firewall.ApplyResult{
		TxnID: tx.ID, Applied: applied,
		StateHash:  hex.EncodeToString(h[:]),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (b *Backend) applyAction(ctx context.Context, a firewall.TxnAction) error {
	if a.Rule == nil {
		return fmt.Errorf("iptables action %q requires a rule", a.Op)
	}
	args, err := ruleToArgs(a.Rule, a.Op)
	if err != nil {
		return err
	}
	out, err := exec.Run(ctx, 20*time.Second, b.tools.Iptables, args...)
	if err != nil {
		return fmt.Errorf("%w: iptables %s: %s", firewall.ErrCommandFailed, a.Op, firstLine(out.Stderr))
	}
	return nil
}

// ruleToArgs converts a normalized rule to iptables argv for the operation.
func ruleToArgs(r *firewall.Rule, op string) ([]string, error) {
	if op != "add" && op != "delete" {
		return nil, fmt.Errorf("iptables adapter supports add/delete, got %q", op)
	}
	chain := directionToChain(r.Direction)
	if chain == "" {
		return nil, fmt.Errorf("unsupported direction %q", r.Direction)
	}
	args := []string{"-I", chain}
	switch op {
	case "add":
		args = append(args, "1") // insert at top for deterministic ordering
	case "delete":
		// delete needs the full match spec without -I/-A prefix
		args = []string{"-D", chain}
	}
	target := "DROP"
	switch r.Action {
	case firewall.ActionAllow:
		target = "ACCEPT"
	case firewall.ActionReject:
		target = "REJECT"
	case firewall.ActionDeny, firewall.ActionDrop:
		target = "DROP"
	case firewall.ActionLog:
		target = "LOG"
	default:
		return nil, fmt.Errorf("unsupported action %q", r.Action)
	}
	if r.Protocol != firewall.ProtoAny && r.Protocol != firewall.ProtoICMP {
		args = append(args, "-p", string(r.Protocol))
	} else if r.Protocol == firewall.ProtoICMP {
		args = append(args, "-p", "icmp")
	}
	if r.Source != "" && r.Source != "any" {
		args = append(args, "-s", r.Source)
	}
	if r.Destination != "" && r.Destination != "any" {
		args = append(args, "-d", r.Destination)
	}
	if r.SourcePort != "" {
		args = appendPortMatch(args, "source", r.SourcePort)
	}
	if r.DestinationPort != "" {
		args = appendPortMatch(args, "destination", r.DestinationPort)
	}
	if r.InterfaceIn != "" {
		args = append(args, "-i", r.InterfaceIn)
	}
	if r.InterfaceOut != "" {
		args = append(args, "-o", r.InterfaceOut)
	}
	if r.State != "" {
		args = append(args, "-m", "conntrack", "--ctstate", r.State)
	}
	if r.Comment != "" {
		args = append(args, "-m", "comment", "--comment", "FYRWALL:RULE:"+r.ID+" "+r.Comment)
	} else {
		args = append(args, "-m", "comment", "--comment", "FYRWALL:RULE:"+r.ID)
	}
	args = append(args, "-j", target)
	return args, nil
}

func normalizePortArg(p string) string {
	// iptables multiport uses comma-separated and colon ranges.
	return strings.ReplaceAll(p, "-", ":")
}

func appendPortMatch(args []string, direction, port string) []string {
	port = normalizePortArg(port)
	flag := "--dport"
	multiFlag := "--dports"
	if direction == "source" {
		flag, multiFlag = "--sport", "--sports"
	}
	if strings.Contains(port, ",") {
		return append(args, "-m", "multiport", multiFlag, port)
	}
	return append(args, flag, port)
}

// Verify compares fresh save output hash to expectation.
func (b *Backend) Verify(ctx context.Context, expected firewall.StateHash) (firewall.VerifyResult, error) {
	out, err := exec.Run(ctx, 15*time.Second, b.tools.IptablesSave)
	if err != nil {
		return firewall.VerifyResult{}, err
	}
	h := sha256.Sum256([]byte(out.Stdout))
	obs := hex.EncodeToString(h[:])
	return firewall.VerifyResult{Match: obs == string(expected), Observed: obs, Expected: string(expected)}, nil
}

// Restore feeds snapshot content back through iptables-restore.
func (b *Backend) Restore(ctx context.Context, snap firewall.Snapshot) error {
	if snap.IptablesSave != "" {
		if err := restore(ctx, b.tools.IptablesRestore, snap.IptablesSave); err != nil {
			return fmt.Errorf("%w: %s", firewall.ErrRollbackFailed, err)
		}
	}
	if snap.IP6TablesSave != "" && b.tools.IP6TablesRestore != "" {
		if err := restore(ctx, b.tools.IP6TablesRestore, snap.IP6TablesSave); err != nil {
			return fmt.Errorf("%w: %s", firewall.ErrRollbackFailed, err)
		}
	}
	return nil
}

func restore(ctx context.Context, bin, content string) error {
	if bin == "" {
		return fmt.Errorf("%s binary not available", bin)
	}
	out, err := exec.RunInput(ctx, 60*time.Second, content, bin)
	if err != nil {
		return fmt.Errorf("%s: %s", bin, firstLine(out.Stderr))
	}
	return nil
}

func (b *Backend) Reload(ctx context.Context) error  { return nil } // iptables has no reload; rules are live
func (b *Backend) Restart(ctx context.Context) error { return nil } // restart handled via service manager

func directionToChain(d firewall.Direction) string {
	switch d {
	case firewall.DirIn:
		return "INPUT"
	case firewall.DirOut:
		return "OUTPUT"
	case firewall.DirForward:
		return "FORWARD"
	}
	return ""
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}
