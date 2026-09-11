// Package auth: strong password policy shared by CLI bootstrap, user
// creation and password change paths (spec sections 13, 35). One policy,
// enforced identically everywhere so a weak password cannot slip in
// through a side door.
package auth

import (
	"fmt"
	"strings"
	"unicode"
)

// PasswordPolicy is the minimum strength bar for every FYRwall password.
type PasswordPolicy struct {
	MinLength        int
	RequireUpper     bool
	RequireLower     bool
	RequireDigit     bool
	RequireSpecial   bool
	MaxConsecutive   int // max identical characters in a row (3 = "aaa" rejected)
	ForbiddenSubstr  []string
}

// DefaultPasswordPolicy is intentionally strict: 14+ chars with upper,
// lower, digit and special. 14 beats the old 12-char rule and stays
// typeable from a password manager.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      14,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		MaxConsecutive: 3,
		ForbiddenSubstr: []string{
			"fyrwall", "potenfyr", "admin", "root", "password",
			"qwerty", "letmein", "iloveyou", "dragon", "monkey",
		},
	}
}

// ValidatePassword checks pw against the policy and returns every reason
// it failed (empty slice = acceptable). Errors are user-presentable.
func ValidatePassword(pw string, p PasswordPolicy) []string {
	var fails []string
	if len(pw) < p.MinLength {
		fails = append(fails, fmt.Sprintf("must be at least %d characters", p.MinLength))
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	var last rune
	var run int
	maxRun := 0
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
		if r == last {
			run++
		} else {
			run = 1
			last = r
		}
		if run > maxRun {
			maxRun = run
		}
	}
	if p.RequireUpper && !hasUpper {
		fails = append(fails, "must contain an uppercase letter")
	}
	if p.RequireLower && !hasLower {
		fails = append(fails, "must contain a lowercase letter")
	}
	if p.RequireDigit && !hasDigit {
		fails = append(fails, "must contain a digit")
	}
	if p.RequireSpecial && !hasSpecial {
		fails = append(fails, "must contain a special character")
	}
	if p.MaxConsecutive > 0 && maxRun > p.MaxConsecutive {
		fails = append(fails, fmt.Sprintf("must not repeat a character more than %d times in a row", p.MaxConsecutive))
	}
	lower := strings.ToLower(pw)
	for _, bad := range p.ForbiddenSubstr {
		if bad != "" && strings.Contains(lower, bad) {
			fails = append(fails, fmt.Sprintf("must not contain %q", bad))
			break
		}
	}
	return fails
}

// ValidatePasswordDefault validates against DefaultPasswordPolicy.
func ValidatePasswordDefault(pw string) []string {
	return ValidatePassword(pw, DefaultPasswordPolicy())
}
