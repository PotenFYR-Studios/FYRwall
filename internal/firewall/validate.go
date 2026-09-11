package firewall

import (
	"fmt"

	"github.com/google/uuid"
)

// ValidateTx runs structural validation plus conflict analysis for a
// transaction against the current rule list. Shared by all backends.
func ValidateTx(tx Transaction, current []Rule) ValidationResult {
	var errs []string
	var conflicts []Conflict

	for _, a := range tx.Actions {
		if a.Rule == nil {
			if a.Op != "delete" && a.Op != "set_policy" {
				errs = append(errs, fmt.Sprintf("action %q requires a rule", a.Op))
			}
			continue
		}
		if e := ValidateRule(a.Rule); len(e) > 0 {
			errs = append(errs, e...)
		}
		switch a.Op {
		case "add":
			conflicts = append(conflicts, AnalyzeConflicts(current, a.Rule)...)
		case "update", "enable", "disable":
			var without []Rule
			for _, r := range current {
				if a.RuleID != "" && r.ID == a.RuleID {
					continue
				}
				if a.Rule != nil && r.ID == a.Rule.ID {
					continue
				}
				without = append(without, r)
			}
			conflicts = append(conflicts, AnalyzeConflicts(without, a.Rule)...)
		}
	}
	conflicts = dedupe(conflicts)
	// Critical conflicts block the transaction outright (spec section 18).
	for _, c := range conflicts {
		if c.Severity == "CRITICAL" {
			errs = append(errs, c.Summary)
		}
	}
	return ValidationResult{Valid: len(errs) == 0, Errors: errs, Conflict: conflicts}
}

// newID returns a fresh UUID string for snapshots and transactions.
func newID() string { return uuid.NewString() }
