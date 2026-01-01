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
	t.Run("all allowed and all required", func(t *testing.T) {
		got, err := parseRequirement(
			true,
			true,
			true,
			true,
			"upper=1,lower=2,symbol=2,digit=3",
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := RequireConfig{
			UpperCharGroup:  1,
			LowerCharGroup:  2,
			SymbolCharGroup: 2,
			DigitCharGroup:  3,
		}

		requirementEqual(t, got, want)
	})

	t.Run("all allowed and none required", func(t *testing.T) {
		got, err := parseRequirement(
			true,
			true,
			true,
			true,
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := RequireConfig{}
		requirementEqual(t, got, want)
	})

	t.Run("none allowed and none required", func(t *testing.T) {
		got, err := parseRequirement(
			false,
			false,
			false,
			false,
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := RequireConfig{}
		requirementEqual(t, got, want)
	})

	t.Run("none allowed but all required", func(t *testing.T) {
		_, err := parseRequirement(
			false,
			false,
			false,
			false,
			"upper=1,lower=2,symbol=2,digit=3",
		)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
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
