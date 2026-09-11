package firewall

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// AnalyzeConflicts runs pairwise and policy-level conflict analysis over the
// rule list plus a candidate rule (candidate may be nil for pure analysis).
// Severity follows spec section 18: INFO, WARNING, HIGH, CRITICAL.
func AnalyzeConflicts(existing []Rule, candidate *Rule) []Conflict {
	var out []Conflict

	rules := make([]Rule, len(existing))
	copy(rules, existing)
	if candidate != nil {
		rules = append(rules, *candidate)
	}

	// Index active rules by priority (order matters for shadowing).
	ordered := make([]Rule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled {
			ordered = append(ordered, r)
		}
	}
	// Stable sort by priority.
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && ordered[j].Priority < ordered[j-1].Priority; j-- {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
		}
	}

	for i := range ordered {
		for j := i + 1; j < len(ordered); j++ {
			out = append(out, pairwise(ordered[i], ordered[j])...)
		}
	}

	if candidate != nil {
		out = append(out, lockoutChecks(existing, candidate)...)
	}
	return dedupe(out)
}

// pairwise detects duplicate, equivalent, shadowed and overlapping rules.
func pairwise(a, b Rule) []Conflict {
	var out []Conflict
	if a.ID == b.ID {
		return out
	}

	switch {
	case sameRule(a, b):
		out = append(out, Conflict{
			Code: "DUPLICATE_RULE", Severity: "WARNING",
			Summary:     fmt.Sprintf("Rules %s and %s are identical", a.ID, b.ID),
			RuleIDs:     []string{a.ID, b.ID},
			Remediation: "Remove one of the duplicate rules.",
		})
	case sameMatch(a, b) && a.Action != b.Action:
		out = append(out, Conflict{
			Code: "ALLOW_DENY_OVERLAP", Severity: "HIGH",
			Summary:     fmt.Sprintf("Rules %s (%s) and %s (%s) match the same traffic with different actions", a.ID, a.Action, b.ID, b.Action),
			RuleIDs:     []string{a.ID, b.ID},
			Remediation: "Reconcile the two rules so the intended action wins.",
		})
	case shadows(a, b):
		out = append(out, Conflict{
			Code: "RULE_SHADOWED", Severity: "WARNING",
			Summary:     fmt.Sprintf("Rule %s is shadowed by earlier rule %s", b.ID, a.ID),
			Details:     fmt.Sprintf("Rule %s (priority %d, %s) is a subset of rule %s (priority %d, %s) which is evaluated first", b.ID, b.Priority, b.Action, a.ID, a.Priority, a.Action),
			RuleIDs:     []string{a.ID, b.ID},
			Remediation: "Reorder or narrow the shadowed rule if it was meant to be reachable.",
		})
	}
	return out
}

// sameRule: normalized fields identical.
func sameRule(a, b Rule) bool {
	return a.Family == b.Family && a.Direction == b.Direction &&
		a.Action == b.Action && a.Protocol == b.Protocol &&
		a.Source == b.Source && a.SourcePort == b.SourcePort &&
		a.Destination == b.Destination && a.DestinationPort == b.DestinationPort &&
		a.InterfaceIn == b.InterfaceIn && a.InterfaceOut == b.InterfaceOut &&
		a.State == b.State
}

// sameMatch: same selector, action may differ.
func sameMatch(a, b Rule) bool {
	return a.Family == b.Family && a.Direction == b.Direction &&
		a.Protocol == b.Protocol &&
		a.Source == b.Source && a.SourcePort == b.SourcePort &&
		a.Destination == b.Destination && a.DestinationPort == b.DestinationPort &&
		a.InterfaceIn == b.InterfaceIn && a.InterfaceOut == b.InterfaceOut &&
		a.State == b.State
}

// shadows reports whether earlier rule a makes later rule b unreachable.
// a shadows b if every packet b matches is also matched by a and a's action
// terminates (allow/deny/reject/drop; log does not terminate).
func shadows(a, b Rule) bool {
	if a.Action == ActionLog {
		return false
	}
	return matchSubset(b, a)
}

// matchSubset: does rule sub match a subset of what rule super matches?
func matchSubset(sub, super Rule) bool {
	if sub.Direction != super.Direction {
		return false
	}
	if !familySubset(sub.Family, super.Family) {
		return false
	}
	if !protoSubset(sub.Protocol, super.Protocol) {
		return false
	}
	if !addrSubset(sub.Source, super.Source) {
		return false
	}
	if !addrSubset(sub.Destination, super.Destination) {
		return false
	}
	if !portSubset(sub.SourcePort, super.SourcePort) {
		return false
	}
	if !portSubset(sub.DestinationPort, super.DestinationPort) {
		return false
	}
	if super.InterfaceIn != "" && super.InterfaceIn != sub.InterfaceIn {
		return false
	}
	if sub.InterfaceIn != "" && super.InterfaceIn == "" {
		return true // sub more specific
	}
	if super.InterfaceOut != "" && super.InterfaceOut != sub.InterfaceOut {
		return false
	}
	return true
}

func familySubset(sub, super Family) bool {
	if sub == super {
		return true
	}
	// both is a superset of ipv4/ipv6; a specific family is a subset of both.
	if super == FamilyBoth {
		return true
	}
	return false
}

func protoSubset(sub, super Protocol) bool {
	if sub == super {
		return true
	}
	return super == ProtoAny
}

func addrSubset(sub, super string) bool {
	sub = normalizeEmpty(sub)
	super = normalizeEmpty(super)
	if super == "" || super == "any" {
		return true
	}
	if sub == super {
		return true
	}
	// sub CIDR contained in super CIDR/net?
	subIP, subNet, err1 := net.ParseCIDR(sub)
	isSubCIDR := err1 == nil
	_, superNet, err2 := net.ParseCIDR(super)
	isSuperCIDR := err2 == nil

	switch {
	case isSuperCIDR && isSubCIDR:
		return superNet.Contains(subIP)
	case isSuperCIDR:
		ip := net.ParseIP(sub)
		return ip != nil && superNet.Contains(ip)
	case isSubCIDR:
		ip := net.ParseIP(super)
		return ip != nil && subNet.Contains(ip)
	}
	return false
}

func portSubset(sub, super string) bool {
	sub, super = normalizeEmpty(sub), normalizeEmpty(super)
	if super == "" {
		return true
	}
	if sub == "" {
		return false // sub matches all ports, super is specific
	}
	if sub == super {
		return true
	}
	subLo, subHi, ok1 := portRange(sub)
	superLo, superHi, ok2 := portRange(super)
	if !ok1 || !ok2 {
		return false
	}
	return subLo >= superLo && subHi <= superHi
}

func portRange(p string) (int, int, bool) {
	p = strings.ReplaceAll(p, ":", "-")
	if i := strings.IndexByte(p, '-'); i >= 0 {
		lo, e1 := strconv.Atoi(p[:i])
		hi, e2 := strconv.Atoi(p[i+1:])
		return lo, hi, e1 == nil && e2 == nil
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 0, 0, false
	}
	return n, n, true
}

func normalizeEmpty(s string) string {
	if s == "any" || s == "*" || s == "0.0.0.0/0" || s == "::/0" {
		return ""
	}
	return s
}

// lockoutChecks guards the management path (spec sections 18, 20).
func lockoutChecks(existing []Rule, candidate *Rule) []Conflict {
	var out []Conflict
	if candidate.Action == ActionAllow && candidate.Direction == DirIn {
		return out // allows never lock out
	}
	if candidate.Direction != DirIn || (candidate.Action != ActionDeny &&
		candidate.Action != ActionReject && candidate.Action != ActionDrop) {
		return out
	}

	// Does this deny rule cover the loopback / local admin path?
	if addrCovers(candidate.Source, "127.0.0.1") || addrCovers(candidate.Destination, "127.0.0.1") {
		out = append(out, Conflict{
			Code: "MANAGEMENT_PORT_LOCKOUT", Severity: "CRITICAL",
			Summary:     "Deny rule matches loopback traffic and would sever local UI access",
			RuleIDs:     []string{candidate.ID},
			Remediation: "Exclude 127.0.0.0/8 from this rule or add an explicit allow for the management interface first.",
		})
	}

	// SSH detected on 22 or a custom management port: deny that spans all
	// inbound traffic on the SSH port risks lockout.
	for _, r := range existing {
		if r.Action == ActionAllow && r.Direction == DirIn &&
			portIntersects(candidate.DestinationPort, r.DestinationPort) &&
			addrOverlap(candidate.Source, r.Source) {
			out = append(out, Conflict{
				Code: "SSH_LOCKOUT_RISK", Severity: "CRITICAL",
				Summary:     fmt.Sprintf("Deny rule %s overlaps allow rule %s covering the management/SSH path", candidate.ID, r.ID),
				RuleIDs:     []string{candidate.ID, r.ID},
				Remediation: "Ensure the explicit allow rule is evaluated before this deny, or add a scoped exception.",
			})
		}
	}
	return out
}

func addrCovers(spec, ip string) bool {
	if spec == "" || spec == "any" {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if _, netw, err := net.ParseCIDR(spec); err == nil {
		return netw.Contains(parsed)
	}
	return spec == ip
}

func addrOverlap(a, b string) bool {
	a, b = normalizeEmpty(a), normalizeEmpty(b)
	if a == "" || b == "" || a == b {
		return true
	}
	return false
}

func portIntersects(a, b string) bool {
	a, b = normalizeEmpty(a), normalizeEmpty(b)
	if a == "" || b == "" {
		return true
	}
	aLo, aHi, _ := portRange(a)
	bLo, bHi, _ := portRange(b)
	return aLo <= bHi && bLo <= aHi
}

func dedupe(in []Conflict) []Conflict {
	seen := map[string]bool{}
	var out []Conflict
	for _, c := range in {
		key := c.Code + "|" + strings.Join(c.RuleIDs, ",")
		if !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}
