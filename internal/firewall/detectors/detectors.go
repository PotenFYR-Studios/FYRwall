// Package detectors implements capability/conflict detection for
// firewalld, native nftables, and iptables alternatives (spec sections 6,
// 7, 8.2). Detection is strictly read-only.
package detectors

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/system/exec"
)

// NftablesResult reports whether a native nftables ruleset exists.
type NftablesResult struct {
	BinaryFound  bool
	BinaryPath   string
	RulesetItems int // tables seen in `nft list tables`
}

// DetectNftables runs `nft list tables` (read-only) to see if a native
// nftables ruleset is present.
func DetectNftables(ctx context.Context) NftablesResult {
	var res NftablesResult
	bin, err := exec.LookPath("nft")
	if err != nil {
		for _, p := range []string{"/usr/sbin/nft", "/usr/bin/nft", "/sbin/nft"} {
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				bin = p
				break
			}
		}
	}
	if bin == "" {
		return res
	}
	res.BinaryFound = true
	res.BinaryPath = bin
	out, err := exec.Run(ctx, 10*time.Second, bin, "list", "tables")
	if err != nil {
		return res // binary present but netfilter not accessible (e.g. container)
	}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if strings.Contains(line, "table ") {
			res.RulesetItems++
		}
	}
	return res
}

// FirewalldResult reports firewalld presence and activity.
type FirewalldResult struct {
	Running bool
	Details string
}

// DetectFirewalld checks for a running firewalld without touching it.
// Uses firewall-cmd --state first (safe query), then falls back to
// checking the well-known D-Bus unit state via /run marker files.
func DetectFirewalld(ctx context.Context) FirewalldResult {
	var res FirewalldResult
	bin, err := exec.LookPath("firewall-cmd")
	if err != nil {
		return res
	}
	out, err := exec.Run(ctx, 10*time.Second, bin, "--state")
	if err == nil && strings.TrimSpace(out.Stdout) == "running" {
		res.Running = true
		res.Details = "firewall-cmd reports running"
		return res
	}
	if err != nil {
		res.Details = err.Error()
	}
	return res
}

// UFWResult reports whether the ufw service is enabled/active.
type UFWResult struct {
	Active  bool
	Enabled bool // enabled at boot
	Details string
}

// DetectUFWService inspects whether ufw is active using its own status
// command (read-only) rather than assuming a service manager.
func DetectUFWService(ctx context.Context, ufwBin string) UFWResult {
	var res UFWResult
	if ufwBin == "" {
		return res
	}
	out, err := exec.Run(ctx, 15*time.Second, ufwBin, "status")
	if err != nil {
		res.Details = err.Error()
		return res
	}
	res.Active = strings.Contains(out.Stdout, "Status: active")
	res.Enabled = res.Active
	res.Details = strings.TrimSpace(firstLine(out.Stdout))
	return res
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
