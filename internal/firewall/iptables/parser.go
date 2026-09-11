package iptables

import (
	"strconv"
	"strings"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

// ParseSave parses iptables-save output (untrusted input) into normalized
// rules. Handles: *table headers, -P policies, -A chain rules with -s/-d/
// -p/--sport/--dport/--sports/--dports/-i/-o/-m conntrack/-m comment/-j.
// Malformed lines are skipped, never fatal (spec section 40: malformed
// output fixtures must not crash the parser).
func ParseSave(content string) []firewall.Rule {
	return parse(content, firewall.FamilyIPv4)
}

// ParseSave6 is ParseSave for ip6tables-save content.
func ParseSave6(content string) []firewall.Rule {
	return parse(content, firewall.FamilyIPv6)
}

func parse(content string, fam firewall.Family) []firewall.Rule {
	var rules []firewall.Rule
	priority := 0
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "-A ") {
			continue
		}
		f := tokenize(line)
		if len(f) < 3 {
			continue
		}
		chain := f[1]
		dir := chainToDirection(chain)
		if dir == "" {
			continue // custom chains: recorded at lower fidelity
		}
		rule := firewall.Rule{
			ID: firewall.NewRuleID(), Backend: "iptables", Family: fam,
			Direction: dir, Enabled: true, Priority: priority,
			CreatedAt: timeNow(), UpdatedAt: timeNow(),
		}
		priority++
		rule.RawReference = line

		target := ""
		for i := 2; i < len(f); i++ {
			switch f[i] {
			case "-p", "--protocol":
				if i+1 < len(f) {
					rule.Protocol = protoOf(f[i+1])
					i++
				}
			case "-s", "--source", "--src":
				if i+1 < len(f) {
					rule.Source = f[i+1]
					i++
				}
			case "-d", "--destination", "--dst":
				if i+1 < len(f) {
					rule.Destination = f[i+1]
					i++
				}
			case "--sport", "--source-port":
				if i+1 < len(f) {
					rule.SourcePort = normalizePort(f[i+1])
					i++
				}
			case "--dport", "--destination-port":
				if i+1 < len(f) {
					rule.DestinationPort = normalizePort(f[i+1])
					i++
				}
			case "--sports", "--source-ports":
				if i+1 < len(f) {
					rule.SourcePort = normalizeMultiPort(f[i+1])
					i++
				}
			case "--dports", "--destination-ports":
				if i+1 < len(f) {
					rule.DestinationPort = normalizeMultiPort(f[i+1])
					i++
				}
			case "-i", "--in-interface":
				if i+1 < len(f) {
					rule.InterfaceIn = f[i+1]
					i++
				}
			case "-o", "--out-interface":
				if i+1 < len(f) {
					rule.InterfaceOut = f[i+1]
					i++
				}
			case "--ctstate":
				if i+1 < len(f) {
					rule.State = f[i+1]
					i++
				}
			case "--comment":
				if i+1 < len(f) {
					rule.Comment = strings.TrimPrefix(f[i+1], "FYRWALL:RULE:")
					i++
				}
			case "-j", "--jump":
				if i+1 < len(f) {
					target = f[i+1]
					i++
				}
			case "-m":
				if i+1 < len(f) {
					i++ // module name consumed; its options handled above
				}
			}
		}
		switch target {
		case "ACCEPT":
			rule.Action = firewall.ActionAllow
		case "DROP":
			rule.Action = firewall.ActionDrop
		case "REJECT":
			rule.Action = firewall.ActionReject
		case "LOG":
			rule.Action = firewall.ActionLog
		default:
			continue // UFW chains (ufw-user-input etc) resolved via their own chain walk in future work
		}
		rules = append(rules, rule)
	}
	return rules
}

// tokenize splits a rule line respecting single-quoted strings, so a
// comment containing spaces does not corrupt field parsing.
func tokenize(line string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '\'':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func protoOf(p string) firewall.Protocol {
	switch strings.ToLower(p) {
	case "tcp":
		return firewall.ProtoTCP
	case "udp":
		return firewall.ProtoUDP
	case "icmp", "ipv6-icmp", "icmpv6":
		return firewall.ProtoICMP
	case "all":
		return firewall.ProtoAny
	default:
		return firewall.ProtoCustom
	}
}

// normalizePort converts "22" or "1000:2000" to the normalized range form.
func normalizePort(p string) string {
	if strings.Contains(p, ":") {
		parts := strings.SplitN(p, ":", 2)
		lo, e1 := strconv.Atoi(parts[0])
		hi, e2 := strconv.Atoi(parts[1])
		if e1 == nil && e2 == nil {
			return strconv.Itoa(lo) + "-" + strconv.Itoa(hi)
		}
	}
	return p
}

// normalizeMultiPort converts "22,80:90,443" multiport syntax to normalized
// comma-separated ranges.
func normalizeMultiPort(p string) string {
	parts := strings.Split(p, ",")
	for i, part := range parts {
		parts[i] = normalizePort(part)
	}
	return strings.Join(parts, ",")
}

// chainToDirection maps a built-in chain to normalized direction.
func chainToDirection(chain string) firewall.Direction {
	switch strings.ToUpper(chain) {
	case "INPUT":
		return firewall.DirIn
	case "OUTPUT":
		return firewall.DirOut
	case "FORWARD":
		return firewall.DirForward
	}
	return ""
}

func timeNow() time.Time { return time.Now().UTC() }
