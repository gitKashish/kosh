package search

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		// Empty Values
		{"empty both", "", "", 0},
		{"empty a", "", "abc", 3},
		{"empty b", "abc", "", 3},

		// Trivial Operations
		{"identical a and b", "abc", "abc", 0},
		{"one deletion", "abcd", "abc", 1},
		{"one addition", "abc", "abcd", 1},
		{"one substitution", "abd", "abc", 1},

		// Full Replace
		{"full replace same length", "abc", "xyz", 3},
		{"full replace diff length", "abcd", "xy", 4},

		// Structural
		{"symmetry", "kitten", "sitting", 3},
		{"case sensitive", "ABC", "abc", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf(
					"levenshtein(%s, %s) = %d, want %d",
					tt.a, tt.b, got, tt.want,
				)
			}
		})
	}
}
