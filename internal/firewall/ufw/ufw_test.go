package ufw

import (
	"strings"
	"testing"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

// Fixtures mirror `ufw status verbose` and `ufw status numbered` output
// including the (v6) section, LIMIT rules and comments (spec section 40).
const sampleVerbose = `Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)
New profiles: skip`

const sampleNumbered = `Status: active

     To                         Action      From
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 80,443/tcp                 ALLOW IN    Anywhere
[ 3] 22/tcp                     LIMIT IN    Anywhere
[ 4] Anywhere                   DENY IN     203.0.113.0/24
[ 5] 53/udp                     ALLOW IN    192.168.1.0/24
[ 6] 22/tcp                     ALLOW IN    Anywhere (v6)
[ 7] 443/tcp                    ALLOW IN    Anywhere (comment web-front)
`

func TestParseStatusVerbose(t *testing.T) {
	st, err := parseStatusVerbose(sampleVerbose)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Enabled {
		t.Error("expected active status")
	}
	if len(st.DefaultPolicies) != 3 {
		t.Fatalf("expected 3 default policies, got %d", len(st.DefaultPolicies))
	}
	in := st.DefaultPolicies[0]
	if in.Direction != firewall.DirIn || in.Action != firewall.ActionDeny {
		t.Errorf("incoming policy = %+v, want in/deny", in)
	}
	out := st.DefaultPolicies[1]
	if out.Direction != firewall.DirOut || out.Action != firewall.ActionAllow {
		t.Errorf("outgoing policy = %+v, want out/allow", out)
	}
	fwd := st.DefaultPolicies[2]
	if fwd.Direction != firewall.DirForward {
		t.Errorf("routed policy direction = %s", fwd.Direction)
	}
}

func TestParseStatusVerboseInactive(t *testing.T) {
	st, err := parseStatusVerbose("Status: inactive\nDefault: deny (incoming), allow (outgoing), disabled (routed)")
	if err != nil {
		t.Fatal(err)
	}
	if st.Enabled {
		t.Error("expected inactive")
	}
}

func TestParseStatusNumbered(t *testing.T) {
	rules, err := parseStatusNumbered(sampleNumbered)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 7 {
		t.Fatalf("expected 7 rules, got %d", len(rules))
	}
	r1 := rules[0]
	if r1.BackendID != "1" {
		t.Errorf("backend ID = %q", r1.BackendID)
	}
	if r1.Action != firewall.ActionAllow || r1.Direction != firewall.DirIn {
		t.Errorf("rule 1 = %s/%s", r1.Action, r1.Direction)
	}
	if r1.Protocol != firewall.ProtoTCP || r1.DestinationPort != "22" {
		t.Errorf("rule 1 proto/port = %s/%s", r1.Protocol, r1.DestinationPort)
	}
	if rules[1].DestinationPort != "80,443" {
		t.Errorf("rule 2 port = %q", rules[1].DestinationPort)
	}
	if rules[2].Comment != "ufw limit" {
		t.Errorf("rule 3 comment = %q", rules[2].Comment)
	}
	if rules[3].Action != firewall.ActionDeny {
		t.Errorf("rule 4 action = %s", rules[3].Action)
	}
	if rules[3].Source != "203.0.113.0/24" {
		t.Errorf("rule 4 source = %q", rules[3].Source)
	}
	if rules[4].Protocol != firewall.ProtoUDP {
		t.Errorf("rule 5 proto = %s", rules[4].Protocol)
	}
	if rules[5].Family != firewall.FamilyIPv6 {
		t.Errorf("rule 6 family = %s", rules[5].Family)
	}
	if !strings.Contains(rules[6].Comment, "web-front") {
		t.Errorf("rule 7 comment = %q", rules[6].Comment)
	}
}

func TestToUFWArgsAreSeparate(t *testing.T) {
	r := &firewall.Rule{Direction: firewall.DirIn, Action: firewall.ActionAllow, Protocol: firewall.ProtoTCP, DestinationPort: "443"}
	args, err := toUFWArgs(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) < 2 || args[0] != "allow" || args[len(args)-1] != "443" {
		t.Fatalf("unexpected argv: %#v", args)
	}
}

func TestParseStatusNumberedMalformed(t *testing.T) {
	garbage := "[ x] garbage line\n[ 9]\n[ 1] short\nStatus: active\n"
	rules, err := parseStatusNumbered(garbage)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules from garbage, got %d", len(rules))
	}
}

func TestToUFWSpec(t *testing.T) {
	r := &firewall.Rule{
		Direction: firewall.DirIn, Action: firewall.ActionAllow,
		Protocol: firewall.ProtoTCP, Source: "10.0.0.0/8", DestinationPort: "443",
	}
	spec, err := toUFWSpec(r)
	if err != nil {
		t.Fatal(err)
	}
	want := "allow proto tcp from 10.0.0.0/8 to any port 443"
	if spec != want {
		t.Errorf("spec = %q, want %q", spec, want)
	}
}

func TestToUFWSpecOutboundDeny(t *testing.T) {
	r := &firewall.Rule{
		Direction: firewall.DirOut, Action: firewall.ActionDeny,
		Protocol: firewall.ProtoUDP, Destination: "203.0.113.9", DestinationPort: "123",
	}
	spec, err := toUFWSpec(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(spec, "deny out") {
		t.Errorf("spec = %q, want deny out prefix", spec)
	}
}
