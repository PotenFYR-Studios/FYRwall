package update

import (
	"context"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.0.9", 1},
		{"2.0.0", "10.0.0", -1},    // numeric, not lexicographic
		{"v1.0.0", "1.0.0", 0},     // v prefix tolerated
		{"0.1.0-dev", "0.1.0", -1}, // suffix sorts older
		{"0.1.0", "0.1.0-dev", 1},
		{"1.0", "1.0.0", 0},
		{"1.10.0", "1.9.0", 1},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckInvalidURL(t *testing.T) {
	// Offline-safe: a bad URL must error, never panic or hang.
	if _, err := Check(context.Background(), "not-a-url", "0.1.0"); err == nil {
		t.Fatal("expected error for invalid manifest URL")
	}
}
