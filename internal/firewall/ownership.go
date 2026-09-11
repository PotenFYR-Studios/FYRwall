package firewall

// PolicyOwner is who owns the active firewall policy (spec section 7).
type PolicyOwner string

const (
	OwnerNone                PolicyOwner = "NONE"
	OwnerUFW                 PolicyOwner = "UFW"
	OwnerIptablesLegacy      PolicyOwner = "IPTABLES_LEGACY"
	OwnerIptablesNFT         PolicyOwner = "IPTABLES_NFT"
	OwnerFirewalld           PolicyOwner = "FIREWALLD"
	OwnerNftablesNative      PolicyOwner = "NFTABLES_NATIVE"
	OwnerMultipleConflicting PolicyOwner = "MULTIPLE_CONFLICTING"
	OwnerUnknown             PolicyOwner = "UNKNOWN"
)

// OwnershipReport is the result of startup / pre-write ownership detection.
type OwnershipReport struct {
	Owner           PolicyOwner `json:"owner"`
	WritesAllowed   bool        `json:"writes_allowed"`
	Reason          string      `json:"reason,omitempty"`
	Conflicts       []string    `json:"conflicts,omitempty"`
	FirewalldActive bool        `json:"firewalld_active"`
	NftablesRuleset bool        `json:"nftables_ruleset_present"`
	UFWActive       bool        `json:"ufw_active"`
	IptablesMode    string      `json:"iptables_mode"` // legacy|nft|unknown
	Details         string      `json:"details,omitempty"`
}

// ResolveOwnership maps detection signals to a policy owner and a
// write-allowed decision. Never silently manages multiple owners: writes
// are blocked on MULTIPLE_CONFLICTING unless the operator explicitly
// acknowledged the conflict (config firewall.allow_write_on_manager_conflict).
func ResolveOwnership(ufwActive, firewalldActive, nftRuleset bool, iptMode string, allowOnConflict bool) OwnershipReport {
	active := 0
	for _, b := range []bool{ufwActive, firewalldActive, nftRuleset} {
		if b {
			active++
		}
	}

	rep := OwnershipReport{
		UFWActive:       ufwActive,
		FirewalldActive: firewalldActive,
		NftablesRuleset: nftRuleset,
		IptablesMode:    iptMode,
	}

	switch {
	case active == 0:
		rep.Owner = OwnerNone
		rep.WritesAllowed = false
		rep.Reason = "no active firewall manager detected; nothing to manage"
		return rep

	case active > 1:
		rep.Owner = OwnerMultipleConflicting
		if ufwActive && firewalldActive {
			rep.Conflicts = append(rep.Conflicts, "UFW and firewalld are both active")
		}
		if nftRuleset && (ufwActive || firewalldActive) {
			rep.Conflicts = append(rep.Conflicts, "a native nftables ruleset coexists with another manager")
		}
		rep.WritesAllowed = allowOnConflict
		if !allowOnConflict {
			rep.Reason = "multiple independent firewall managers are active; writes blocked until an administrator acknowledges the conflict"
		} else {
			rep.Reason = "conflict acknowledged by administrator; writes enabled for the selected backend only"
		}
		return rep
	}

	switch {
	case ufwActive:
		rep.Owner = OwnerUFW
		rep.WritesAllowed = true
	case firewalldActive:
		rep.Owner = OwnerFirewalld
		rep.WritesAllowed = false
		rep.Reason = "firewalld owns the policy; FYRwall v1 is read-only against firewalld"
	case nftRuleset:
		rep.Owner = OwnerNftablesNative
		rep.WritesAllowed = false
		rep.Reason = "native nftables ruleset present; FYRwall v1 is read-only against nftables"
	default:
		rep.Owner = OwnerUnknown
		rep.WritesAllowed = false
	}
	return rep
}
