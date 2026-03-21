package cmd

import (
	"strings"
	"testing"
)

func TestRandomInt(t *testing.T) {

	t.Run("normal range", func(t *testing.T) {
		n, err := randomInt(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n < 0 || n >= 10 {
			t.Errorf("randomInt(10) = %d, want between 0-9", n)
		}
	})

	t.Run("edge case zero or negative", func(t *testing.T) {
		_, err := randomInt(0)
		if err == nil {
			t.Errorf("expected error for max=0, got nil")
		}

		_, err = randomInt(-5)
		if err == nil {
			t.Errorf("expected error for max=-5, got nil")
		}
	})

	t.Run("edge case max=1", func(t *testing.T) {
		n, err := randomInt(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != 0 {
			t.Errorf("randomInt(1)=%d, want 0", n)
		}
	})
}

func TestRandomChar(t *testing.T) {
	t.Run("normal string", func(t *testing.T) {
		c, err := randomChar("abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains("abc", string(c)) {
			t.Errorf("randomChar(abc)=%s, want a, b or c", string(c))
		}
	})

	t.Run("edge case empty string", func(t *testing.T) {
		_, err := randomChar("")
		if err == nil {
			t.Errorf("expected an error, but got nil")
		}
	})

	t.Run("edge case single character", func(t *testing.T) {
		c, err := randomChar("a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(c) != "a" {
			t.Errorf("randomChar(a)=%s, want a", string(c))
		}
	})
}

func TestParseRequirement(t *testing.T) {
	tests := []struct {
		name        string
		upper       bool
		lower       bool
		digit       bool
		symbol      bool
		requireStr  string
		want        RequireConfig
		expectError bool
	}{
		{
			name:       "empty require string",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "",
			want:       RequireConfig{},
		},
		{
			name:       "single requirement",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "upper=2",
			want: RequireConfig{
				UpperCharGroup: 2,
			},
		},
		{
			name:       "multiple requirements",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "upper=1,lower=2,symbol=2,digit=3",
			want: RequireConfig{
				UpperCharGroup:  1,
				LowerCharGroup:  2,
				SymbolCharGroup: 2,
				DigitCharGroup:  3,
			},
		},
		{
			name:       "zero values are allowed",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "upper=0,lower=0",
			want: RequireConfig{
				UpperCharGroup: 0,
				LowerCharGroup: 0,
			},
		},
		{
			name:        "invalid format missing equals",
			upper:       true,
			lower:       true,
			digit:       true,
			symbol:      true,
			requireStr:  "upper",
			expectError: true,
		},
		{
			name:        "invalid format empty value",
			upper:       true,
			lower:       true,
			digit:       true,
			symbol:      true,
			requireStr:  "upper=",
			expectError: true,
		},
		{
			name:        "non integer value",
			upper:       true,
			lower:       true,
			digit:       true,
			symbol:      true,
			requireStr:  "upper=abc",
			expectError: true,
		},
		{
			name:        "negative value",
			upper:       true,
			lower:       true,
			digit:       true,
			symbol:      true,
			requireStr:  "upper=-1",
			expectError: true,
		},
		{
			name:        "required but not allowed (upper)",
			upper:       false,
			lower:       true,
			digit:       true,
			symbol:      true,
			requireStr:  "upper=1",
			expectError: true,
		},
		{
			name:       "unknown group is rejected implicitly",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "foo=2",
			want: RequireConfig{
				CharGroup("foo"): 2,
			},
		},
		{
			name:       "duplicate keys last wins",
			upper:      true,
			lower:      true,
			digit:      true,
			symbol:     true,
			requireStr: "upper=1,upper=3",
			want: RequireConfig{
				UpperCharGroup: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRequirement(
				tt.upper,
				tt.lower,
				tt.digit,
				tt.symbol,
				tt.requireStr,
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			requirementEqual(t, got, tt.want)
		})
	}
}

// Helpers
func requirementEqual(t *testing.T, got, want RequireConfig) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("map length: got %d, want %d", len(got), len(want))
	}

	for group, wantVal := range want {
		gotVal, ok := got[group]
		if !ok {
			t.Errorf("key '%s' absent in result", group)
		} else if gotVal != wantVal {
			t.Errorf("%s: got %d, want %d", group, gotVal, wantVal)
		}
	}

	// check for unexpected values
	for group := range got {
		if _, ok := want[group]; !ok {
			t.Errorf("unexpected key '%s' in result", group)
		}
	}
}
