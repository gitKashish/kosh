package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.plutolab.org/plutolab/kosh/internal/model"
)

func TestSanitizeProfileName(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		clean       string
		expectError bool
	}{
		{
			name:        "profile should only have alphanumeric, underscore and hyphen",
			raw:         "My_profile $100",
			clean:       "My_profile_100",
			expectError: false,
		},
		{
			name:        "leading and trailing whitespace are trimmed",
			raw:         "   work-white-space  ",
			clean:       "work-white-space",
			expectError: false,
		},
		{
			name:        "leading and trailing underscore or hyphen",
			raw:         "--YourProfile__",
			clean:       "YourProfile",
			expectError: false,
		},
		{
			name:        "multiple continuous underscore converted to one",
			raw:         "work__main",
			clean:       "work_main",
			expectError: false,
		},
		{
			name:        "multiple whitespaces converted into single underscore",
			raw:         "work   main",
			clean:       "work_main",
			expectError: false,
		},
		{
			name:        "empty string must return error",
			raw:         "",
			clean:       "",
			expectError: true,
		},
		{
			name:        "sanitized empty profile name must return error",
			raw:         "$__##  ",
			clean:       "",
			expectError: true,
		},
		{
			name:        "00 reserved windows file names are invalid profile names",
			raw:         "nul",
			clean:       "",
			expectError: true,
		},
		{
			name:        "01 reserved windows file names are invalid profile names",
			raw:         "lpt2",
			clean:       "",
			expectError: true,
		},
		{
			name:        "02 reserved windows file names are invalid profile names",
			raw:         "__lpt2  ",
			clean:       "",
			expectError: true,
		},
		{
			name:        "03 partial match with reserved should be allowed",
			raw:         "lpt23",
			clean:       "lpt23",
			expectError: false,
		},
		{
			name:        "name is truncated to fit the filename length limit",
			raw:         strings.Repeat("a", 300),
			clean:       strings.Repeat("a", 252),
			expectError: false,
		},
		{
			name:        "truncation does not leave a trailing separator",
			raw:         strings.Repeat("a", 251) + "_bcd",
			clean:       strings.Repeat("a", 251),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := SanitizeProfileName(tt.raw)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if c != tt.clean {
				t.Errorf("profile name %s, must be %s", c, tt.clean)
			}
		})
	}
}

// newTestProfiles points the profile service at a throwaway home directory and
// creates a profile file for each name given.
func newTestProfiles(t *testing.T, names ...string) *KoshProfile {
	t.Helper()

	home := t.TempDir()
	// os.UserHomeDir reads USERPROFILE on Windows and HOME elsewhere.
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".kosh", "profiles")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("create profiles dir: %v", err)
	}
	for _, name := range names {
		path := filepath.Join(dir, name+".db")
		if err := os.WriteFile(path, []byte("vault"), 0600); err != nil {
			t.Fatalf("create profile %q: %v", name, err)
		}
	}

	return NewProfileService()
}

func TestResolveProfileIgnoresCase(t *testing.T) {
	// A profile keeps the casing it was created with, but any spelling has to
	// find it: two profiles differing only in case would collide the moment the
	// profiles directory is copied to a case-insensitive filesystem, so they
	// must be refused on every platform - not only where the filesystem says so.
	tests := []struct {
		name     string
		lookup   string
		expected model.Profile
		exists   bool
	}{
		{name: "exact spelling", lookup: "Work", expected: "Work", exists: true},
		{name: "lowercased spelling", lookup: "work", expected: "Work", exists: true},
		{name: "uppercased spelling", lookup: "WORK", expected: "Work", exists: true},
		{name: "surrounding whitespace is ignored", lookup: "  work  ", expected: "Work", exists: true},
		{name: "unrelated name does not resolve", lookup: "personal", expected: "", exists: false},
		{name: "empty name does not resolve", lookup: "", expected: "", exists: false},
		{name: "partial name does not resolve", lookup: "wor", expected: "", exists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProfiles(t, "Work")

			stored, exists, err := p.ResolveProfile(tt.lookup)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tt.exists {
				t.Errorf("exists = %v, want %v", exists, tt.exists)
			}
			// The stored spelling is what callers open and delete by, so a
			// lookup that folds case must still hand back the name on disk.
			if stored != tt.expected {
				t.Errorf("resolved to %q, want %q", stored, tt.expected)
			}
		})
	}
}

func TestResolveProfilePrefersExactSpelling(t *testing.T) {
	// A directory carried over from a case-sensitive machine can hold both
	// spellings at once. Only such a filesystem can stage that here: elsewhere
	// the second file is the first one, and there is no ambiguity to resolve.
	p := newTestProfiles(t, "work", "Work")

	stored, err := p.LoadProfiles("work", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) < 2 {
		t.Skip("filesystem is case-insensitive; both spellings are one file")
	}

	for _, lookup := range []string{"work", "Work"} {
		stored, exists, err := p.ResolveProfile(lookup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatalf("%q did not resolve", lookup)
		}
		if string(stored) != lookup {
			t.Errorf("%q resolved to %q, want the exact spelling", lookup, stored)
		}
	}
}

func TestResolveProfileRejectsNamesOutsideProfilesDir(t *testing.T) {
	// A profile is an entry of the profiles directory. A name reaching anywhere
	// else names no profile - it must not resolve, or `kosh profile delete`
	// would go on to scrub whatever file it landed on.
	p := newTestProfiles(t, "work")

	home := t.TempDir()
	outside := filepath.Join(home, "outside.db")
	if err := os.WriteFile(outside, []byte("do not touch"), 0600); err != nil {
		t.Fatal(err)
	}

	lookups := []string{
		"../outside",
		"../../outside",
		filepath.Join(home, "outside"),
		"work:stream",
	}

	for _, lookup := range lookups {
		t.Run(lookup, func(t *testing.T) {
			stored, exists, err := p.ResolveProfile(lookup)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists {
				t.Errorf("%q resolved to %q, want no match", lookup, stored)
			}
		})
	}

	// The file the traversal pointed at is still there and untouched.
	if content, err := os.ReadFile(outside); err != nil {
		t.Errorf("target file disturbed: %v", err)
	} else if string(content) != "do not touch" {
		t.Errorf("target file content = %q, want it unchanged", content)
	}
}

func TestLoadProfiles(t *testing.T) {
	tests := []struct {
		name       string
		filter     string
		exactMatch bool
		expected   []model.Profile
	}{
		{
			name:     "empty filter lists every profile",
			filter:   "",
			expected: []model.Profile{"Work", "personal"},
		},
		{
			name:     "filter matches regardless of case",
			filter:   "WORK",
			expected: []model.Profile{"Work"},
		},
		{
			name:     "filter matches on a substring",
			filter:   "ersona",
			expected: []model.Profile{"personal"},
		},
		{
			name:       "exact match ignores case but not partial names",
			filter:     "work",
			exactMatch: true,
			expected:   []model.Profile{"Work"},
		},
		{
			name:       "exact match rejects a partial name",
			filter:     "wor",
			exactMatch: true,
			expected:   []model.Profile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProfiles(t, "Work", "personal")

			profiles, err := p.LoadProfiles(tt.filter, tt.exactMatch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(profiles, tt.expected) {
				t.Errorf("profiles = %v, want %v", profiles, tt.expected)
			}
		})
	}
}

func TestLoadProfilesIgnoresNonProfileEntries(t *testing.T) {
	p := newTestProfiles(t, "work")

	dir, err := p.GetProfilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "nested.db"), 0700); err != nil {
		t.Fatal(err)
	}

	profiles, err := p.LoadProfiles("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(profiles, []model.Profile{"work"}) {
		t.Errorf("profiles = %v, want [work]", profiles)
	}
}

func TestLoadProfilesWithoutProfilesDirectory(t *testing.T) {
	// A fresh install has no profiles directory yet. That is an empty list, not
	// a failure - every command starts by asking which profiles exist.
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	profiles, err := NewProfileService().LoadProfiles("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("profiles = %v, want none", profiles)
	}
}
