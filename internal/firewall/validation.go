package firewall

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ValidateRule statically checks one normalized rule for structural errors
// (malformed IP/CIDR, invalid ports, protocol/family mismatches, etc.).
// Returns a list of human-readable error strings; empty means valid.
func ValidateRule(r *Rule) []string {
	var errs []string

	switch r.Family {
	case FamilyIPv4, FamilyIPv6, FamilyBoth:
	default:
		errs = append(errs, fmt.Sprintf("invalid family %q", r.Family))
	}
	switch r.Direction {
	case DirIn, DirOut, DirForward:
	default:
		errs = append(errs, fmt.Sprintf("invalid direction %q", r.Direction))
	}
	switch r.Action {
	case ActionAllow, ActionDeny, ActionReject, ActionDrop, ActionLog:
	default:
		errs = append(errs, fmt.Sprintf("invalid action %q", r.Action))
	}
	switch r.Protocol {
	case ProtoAny, ProtoTCP, ProtoUDP, ProtoICMP, ProtoCustom:
	default:
		errs = append(errs, fmt.Sprintf("invalid protocol %q", r.Protocol))
	}

	if r.Source != "" && r.Source != "any" {
		if e := validateAddr(r.Source, r.Family); e != "" {
			errs = append(errs, "source: "+e)
		}
	}
	if r.Destination != "" && r.Destination != "any" {
		if e := validateAddr(r.Destination, r.Family); e != "" {
			errs = append(errs, "destination: "+e)
		}
	}
	if e := validatePortField(r.SourcePort); e != "" {
		errs = append(errs, "source_port: "+e)
	}
	if e := validatePortField(r.DestinationPort); e != "" {
		errs = append(errs, "destination_port: "+e)
	}

	// Port fields only make sense for port-capable protocols.
	if (r.SourcePort != "" || r.DestinationPort != "") &&
		r.Protocol != ProtoAny && r.Protocol != ProtoTCP && r.Protocol != ProtoUDP && r.Protocol != ProtoCustom {
		errs = append(errs, fmt.Sprintf("ports set but protocol %q does not use ports", r.Protocol))
	}

	// Interface names: only alphanumerics, dot, dash, at-sign, colon (VLAN).
	for _, ifn := range []struct{ name, val string }{{"interface_in", r.InterfaceIn}, {"interface_out", r.InterfaceOut}} {
		if ifn.val == "" {
			continue
		}
		if len(ifn.val) > 15 {
			errs = append(errs, fmt.Sprintf("%s exceeds IFNAMSIZ (15): %q", ifn.name, ifn.val))
		}
		for _, c := range ifn.val {
			ok := c == '.' || c == '-' || c == '@' || c == ':' ||
				(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			if !ok {
				errs = append(errs, fmt.Sprintf("%s has invalid character %q", ifn.name, string(c)))
				break
			}
		}
	}

	// Loopback protection: blocking loopback traffic breaks local IPC.
	if r.Direction == DirIn && r.Action != ActionAllow && r.Action != ActionLog {
		if isLoopbackSpec(r.Source) || isLoopbackSpec(r.Destination) {
			errs = append(errs, "blocking rule on loopback interface would break local IPC (unsafe)")
		}
	}
	return errs
}

func validateAddr(s string, fam Family) string {
	if strings.Contains(s, "/") {
		ip, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			return fmt.Sprintf("invalid CIDR %q", s)
		}
		return checkFamily(ip, ipnet.String(), fam)
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Sprintf("invalid IP %q", s)
	}
	return checkFamily(ip, s, fam)
}

func checkFamily(ip net.IP, display string, fam Family) string {
	is4 := ip.To4() != nil
	switch fam {
	case FamilyIPv4:
		if !is4 {
			return fmt.Sprintf("IPv6 address %q with family ipv4", display)
		}
	case FamilyIPv6:
		if is4 {
			return fmt.Sprintf("IPv4 address %q with family ipv6", display)
		}
	}
	return ""
}

func validatePortField(p string) string {
	if p == "" {
		return ""
	}
	if strings.Contains(p, "-") || strings.Contains(p, ":") {
		sep := "-"
		if strings.Contains(p, ":") && !strings.Contains(p, "-") {
			sep = ":"
		}
		parts := strings.SplitN(p, sep, 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return fmt.Sprintf("invalid port range %q", p)
		}
		if lo < 1 || hi > 65535 || lo > hi {
			return fmt.Sprintf("invalid port range %d-%d", lo, hi)
		}
		return ""
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Sprintf("invalid port %q", p)
	}
	return ""
}

func isLoopbackSpec(addr string) bool {
	if addr == "" || addr == "any" {
		return false
	}
	if strings.HasPrefix(addr, "127.") || addr == "::1" || addr == "lo" {
		return true
	}
	if ip := net.ParseIP(strings.Split(addr, "/")[0]); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
