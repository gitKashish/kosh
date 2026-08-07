package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// vaultContents stands in for a profile database: long enough that random
// replacement bytes cannot plausibly reproduce it.
var vaultContents = []byte("SQLite format 3\x00" + "s3cr3t-github-token" + "correct horse battery staple")

// writeTempFile creates a file with the given contents and returns its path.
func writeTempFile(t *testing.T, contents []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "default.db")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatalf("unexpected error writing the fixture: %v", err)
	}
	return path
}

// readFile returns the contents on disk.
func readFile(t *testing.T, path string) []byte {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}
	return contents
}

// Test secure delete
func TestOverwriteFile(t *testing.T) {
	t.Run("replaces the file contents", func(t *testing.T) {
		path := writeTempFile(t, vaultContents)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := readFile(t, path); bytes.Equal(got, vaultContents) {
			t.Error("file was not scrubbed, contents are unchanged")
		}
	})

	t.Run("leaves no trace of the original secret", func(t *testing.T) {
		path := writeTempFile(t, vaultContents)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := readFile(t, path)
		for _, secret := range [][]byte{
			[]byte("s3cr3t-github-token"),
			[]byte("correct horse battery staple"),
		} {
			if bytes.Contains(got, secret) {
				t.Errorf("scrubbed file still contains %q", secret)
			}
		}
	})

	t.Run("preserves the file size", func(t *testing.T) {
		path := writeTempFile(t, vaultContents)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got, want := len(readFile(t, path)), len(vaultContents); got != want {
			t.Errorf("file size = %d, want %d", got, want)
		}
	})

	t.Run("scrubs without removing the file", func(t *testing.T) {
		// removal is the caller's job, in core.KoshProfile.DeleteProfile
		path := writeTempFile(t, vaultContents)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(path); err != nil {
			t.Errorf("file should still exist after scrubbing: %v", err)
		}
	})

	t.Run("scrubbing twice keeps working", func(t *testing.T) {
		path := writeTempFile(t, vaultContents)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error on the first pass: %v", err)
		}
		first := readFile(t, path)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error on the second pass: %v", err)
		}

		if bytes.Equal(readFile(t, path), first) {
			t.Error("the second pass wrote the same bytes as the first")
		}
	})

	t.Run("an empty file is left alone without error", func(t *testing.T) {
		path := writeTempFile(t, nil)

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := readFile(t, path); len(got) != 0 {
			t.Errorf("empty file grew to %d bytes", len(got))
		}
	})

	t.Run("a single byte file is scrubbed", func(t *testing.T) {
		// the smallest input that still takes the size > 0 branch
		path := writeTempFile(t, []byte{0xFF})

		if err := OverwriteFile(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := len(readFile(t, path)); got != 1 {
			t.Errorf("file size = %d, want 1", got)
		}
	})

	t.Run("a missing file returns an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "no-such-profile.db")

		err := OverwriteFile(path)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !os.IsNotExist(err) {
			t.Errorf("expected a not-exist error, got %v", err)
		}
	})
}
