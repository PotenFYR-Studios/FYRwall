package firewall

import (
	"testing"
)

func mkRule(id string, fam Family, dir Direction, act Action, proto Protocol, src, dst, dport string, prio int) Rule {
	return Rule{
		ID: id, Family: fam, Direction: dir, Action: act, Protocol: proto,
		Source: src, Destination: dst, DestinationPort: dport, Enabled: true, Priority: prio,
	}
}

func TestValidateRuleAcceptsValid(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "192.168.1.0/24", "", "443", 1)
	if errs := ValidateRule(&r); len(errs) != 0 {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
}

func TestValidateRuleRejectsBadCIDR(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "192.168.1.999/24", "", "", 1)
	errs := ValidateRule(&r)
	if len(errs) == 0 {
		t.Fatal("expected error for malformed CIDR")
	}
}

func TestValidateRuleRejectsFamilyMismatch(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "fe80::1", "", "", 1)
	errs := ValidateRule(&r)
	if len(errs) == 0 {
		t.Fatal("expected IPv6 address under ipv4 family to fail")
	}
}

func TestValidateRuleRejectsBadPort(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "99999", 1)
	errs := ValidateRule(&r)
	if len(errs) == 0 {
		t.Fatal("expected port 99999 to fail")
	}
	r2 := mkRule("r2", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "10-5", 1)
	if errs := ValidateRule(&r2); len(errs) == 0 {
		t.Fatal("expected inverted port range to fail")
	}
}

func TestValidateRuleRejectsPortsOnICMP(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoICMP, "", "", "22", 1)
	if errs := ValidateRule(&r); len(errs) == 0 {
		t.Fatal("expected ports on icmp to fail")
	}
}

func TestValidateRuleRejectsLoopbackBlock(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionDeny, ProtoAny, "127.0.0.1", "", "", 1)
	errs := ValidateRule(&r)
	found := false
	for _, e := range errs {
		if len(e) > 8 && e[:8] == "blocking" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected loopback block warning, got %v", errs)
	}
}

func TestValidateRuleRejectsBadInterface(t *testing.T) {
	r := mkRule("r1", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "", 1)
	r.InterfaceIn = "eth0; rm -rf /"
	if errs := ValidateRule(&r); len(errs) == 0 {
		t.Fatal("expected shell metacharacters in interface name to fail")
	}
}

func TestAnalyzeConflictsDuplicate(t *testing.T) {
	a := mkRule("a", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "10.0.0.0/8", "", "22", 1)
	b := mkRule("b", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "10.0.0.0/8", "", "22", 2)
	conflicts := AnalyzeConflicts([]Rule{a}, &b)
	found := false
	for _, c := range conflicts {
		if c.Code == "DUPLICATE_RULE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate detection, got %+v", conflicts)
	}
}

func TestAnalyzeConflictsShadowed(t *testing.T) {
	broad := mkRule("broad", FamilyIPv4, DirIn, ActionDeny, ProtoAny, "", "", "", 1)
	narrow := mkRule("narrow", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "10.0.0.5", "", "22", 2)
	conflicts := AnalyzeConflicts([]Rule{broad}, &narrow)
	found := false
	for _, c := range conflicts {
		if c.Code == "RULE_SHADOWED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected shadow detection, got %+v", conflicts)
	}
}

func TestAnalyzeConflictsAllowDenyOverlap(t *testing.T) {
	a := mkRule("a", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "443", 1)
	b := mkRule("b", FamilyIPv4, DirIn, ActionDeny, ProtoTCP, "", "", "443", 2)
	conflicts := AnalyzeConflicts([]Rule{a}, &b)
	found := false
	for _, c := range conflicts {
		if c.Code == "ALLOW_DENY_OVERLAP" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected allow/deny overlap, got %+v", conflicts)
	}
}

func TestAnalyzeConflictsSSHLockout(t *testing.T) {
	sshAllow := mkRule("ssh", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "22", 1)
	broadDeny := mkRule("deny", FamilyIPv4, DirIn, ActionDeny, ProtoAny, "", "", "", 2)
	conflicts := AnalyzeConflicts([]Rule{sshAllow}, &broadDeny)
	found := false
	for _, c := range conflicts {
		if c.Code == "SSH_LOCKOUT_RISK" && c.Severity == "CRITICAL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected SSH lockout critical, got %+v", conflicts)
	}
}

func TestAnalyzeConflictsLoopbackCritical(t *testing.T) {
	loopDeny := mkRule("ld", FamilyIPv4, DirIn, ActionDeny, ProtoAny, "127.0.0.0/8", "", "", 1)
	conflicts := AnalyzeConflicts(nil, &loopDeny)
	found := false
	for _, c := range conflicts {
		if c.Code == "MANAGEMENT_PORT_LOCKOUT" && c.Severity == "CRITICAL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected management lockout critical, got %+v", conflicts)
	}
}

func TestValidateTxRejectsCriticalConflict(t *testing.T) {
	sshAllow := mkRule("ssh", FamilyIPv4, DirIn, ActionAllow, ProtoTCP, "", "", "22", 1)
	tx := Transaction{
		ID: "t1", Backend: "ufw",
		Actions: []TxnAction{{Op: "add", Rule: func() *Rule {
			r := mkRule("deny-all", FamilyIPv4, DirIn, ActionDeny, ProtoAny, "", "", "", 5)
			return &r
		}()}},
	}
	vr := ValidateTx(tx, []Rule{sshAllow})
	if vr.Valid {
		t.Fatal("expected transaction with critical conflict to be invalid")
	}
}

func TestResolveOwnership(t *testing.T) {
	cases := []struct {
		ufw, fwalld, nft bool
		mode             string
		want             PolicyOwner
		writes           bool
	}{
		{false, false, false, "legacy", OwnerNone, false},
		{true, false, false, "nft", OwnerUFW, true},
		{false, true, false, "nft", OwnerFirewalld, false},
		{false, false, true, "nft", OwnerNftablesNative, false},
		{true, true, false, "nft", OwnerMultipleConflicting, false},
		{true, true, false, "nft", OwnerMultipleConflicting, true}, // acknowledged
	}
	for i, tc := range cases {
		got := ResolveOwnership(tc.ufw, tc.fwalld, tc.nft, tc.mode, tc.writes && tc.fwalld)
		if got.Owner != tc.want {
			t.Errorf("case %d: owner = %s, want %s", i, got.Owner, tc.want)
		}
		if tc.want == OwnerMultipleConflicting {
			if got.WritesAllowed != tc.writes {
				t.Errorf("case %d: writes_allowed = %v, want %v", i, got.WritesAllowed, tc.writes)
			}
		}
	}
}
