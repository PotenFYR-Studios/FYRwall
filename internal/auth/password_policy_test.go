package auth

import "testing"

func TestValidatePasswordAcceptsStrong(t *testing.T) {
	pw := "Tide-Market7/lamp.Post"
	if fails := ValidatePasswordDefault(pw); len(fails) != 0 {
		t.Fatalf("expected strong password to pass, got: %v", fails)
	}
}

func TestValidatePasswordRejectsWeak(t *testing.T) {
	cases := map[string]string{
		"short1!Aa":            "characters",   // too short
		"alllowercase123!x":    "uppercase",    // no upper
		"ALLUPPERCASE123!X":    "lowercase",    // no lower
		"NoDigitsHere!aaaa":    "digit",        // no digit
		"NoSpecialChar1234Abcd": "special",     // no special
		"aaaaaaa1!AAAAA":       "repeat",       // repeated run
		"MyFyrwallPass1!x":     "contain",      // forbidden substring
	}
	for pw, want := range cases {
		fails := ValidatePasswordDefault(pw)
		if len(fails) == 0 {
			t.Errorf("%q: expected rejection", pw)
			continue
		}
		found := false
		for _, f := range fails {
			if contains(f, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: expected failure mentioning %q, got %v", pw, want, fails)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
