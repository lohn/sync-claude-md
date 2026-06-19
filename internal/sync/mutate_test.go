package sync

import (
	"os"
	"testing"
)

// refCases drives the mutate tests over every target's reference line so that
// adding an agent does not require new tests — only a new entry here.
var refCases = []struct {
	name string
	ref  string
}{
	{name: "claude", ref: claudeTarget.ref},
	{name: "gemini", ref: geminiTarget.ref},
}

// TestSyncFileCreates writes a fresh target file containing only the reference.
func TestSyncFileCreates(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			modified, err := syncFile("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("syncFile failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true")
			}
			if got := readFile(t, "TARGET.md"); got != rc.ref+"\n" {
				t.Fatalf("got %q, want %q", got, rc.ref+"\n")
			}
		})
	}
}

// TestSyncFileCheckModeDoesNotCreate reports drift without writing the file.
func TestSyncFileCheckModeDoesNotCreate(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			modified, err := syncFile("TARGET.md", rc.ref, true)
			if err != nil {
				t.Fatalf("syncFile failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true in check mode")
			}
			if _, err := os.Stat("TARGET.md"); !os.IsNotExist(err) {
				t.Fatal("expected TARGET.md to NOT be created in check mode")
			}
		})
	}
}

// TestUpdateTargetPrepends inserts the reference above existing content,
// separated by a single blank line.
func TestUpdateTargetPrepends(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			writeFile(t, "TARGET.md", "# Existing\n")

			modified, err := updateTarget("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("updateTarget failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true")
			}
			want := rc.ref + "\n\n# Existing\n"
			if got := readFile(t, "TARGET.md"); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

// TestUpdateTargetIdempotent leaves the file untouched when the reference is
// already present anywhere in the file.
func TestUpdateTargetIdempotent(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			original := "# Title\n\nintro\n\n" + rc.ref + "\n"
			writeFile(t, "TARGET.md", original)

			modified, err := updateTarget("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("updateTarget failed: %v", err)
			}
			if modified {
				t.Fatal("expected modified=false when reference already present")
			}
			if got := readFile(t, "TARGET.md"); got != original {
				t.Fatalf("content modified unexpectedly: %q", got)
			}
		})
	}
}

// TestRemoveRefKeepsContent strips the leading reference but preserves the rest.
func TestRemoveRefKeepsContent(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			writeFile(t, "TARGET.md", rc.ref+"\n\n# Existing\n")

			modified, err := removeRef("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("removeRef failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true")
			}
			if got := readFile(t, "TARGET.md"); got != "# Existing\n" {
				t.Fatalf("got %q, want %q", got, "# Existing\n")
			}
		})
	}
}

// TestRemoveRefDeletesEmptyFile removes the file when only the reference remained.
func TestRemoveRefDeletesEmptyFile(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			writeFile(t, "TARGET.md", rc.ref+"\n")

			modified, err := removeRef("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("removeRef failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true")
			}
			if _, err := os.Stat("TARGET.md"); !os.IsNotExist(err) {
				t.Fatal("expected TARGET.md to be deleted")
			}
		})
	}
}

// TestRemoveRefRemovesMovedReference strips the reference even when the user has
// moved it below other content, mirroring updateTarget's "present anywhere" rule.
func TestRemoveRefRemovesMovedReference(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			writeFile(t, "TARGET.md", "# Title\n\n"+rc.ref+"\n")

			modified, err := removeRef("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("removeRef failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true for moved reference")
			}
			if got := readFile(t, "TARGET.md"); got != "# Title\n" {
				t.Fatalf("got %q, want %q", got, "# Title\n")
			}
		})
	}
}

// TestRemoveRefIgnoresInlineSubstring leaves lines that merely contain the
// reference as a substring untouched; only standalone reference lines are removed.
func TestRemoveRefIgnoresInlineSubstring(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			original := "See " + rc.ref + " for details.\n"
			writeFile(t, "TARGET.md", original)

			modified, err := removeRef("TARGET.md", rc.ref, false)
			if err != nil {
				t.Fatalf("removeRef failed: %v", err)
			}
			if modified {
				t.Fatal("expected modified=false for substring-only reference")
			}
			if got := readFile(t, "TARGET.md"); got != original {
				t.Fatalf("content modified unexpectedly: %q", got)
			}
		})
	}
}

// TestRemoveRefMissingFileNoOps returns (false, nil) when the target file does
// not exist, so cleanup of a deleted AGENTS.md never fails for a target that
// was never synced in that directory.
func TestRemoveRefMissingFileNoOps(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			modified, err := removeRef("MISSING.md", rc.ref, false)
			if err != nil {
				t.Fatalf("expected no error for missing file, got: %v", err)
			}
			if modified {
				t.Fatal("expected modified=false for missing file")
			}
		})
	}
}

// TestRemoveRefCheckMode reports removal without writing.
func TestRemoveRefCheckMode(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			original := rc.ref + "\n"
			writeFile(t, "TARGET.md", original)

			modified, err := removeRef("TARGET.md", rc.ref, true)
			if err != nil {
				t.Fatalf("removeRef failed: %v", err)
			}
			if !modified {
				t.Fatal("expected modified=true in check mode")
			}
			if got := readFile(t, "TARGET.md"); got != original {
				t.Fatalf("file changed in check mode: %q", got)
			}
		})
	}
}

// TestRemoveRefDoesNotMatchOtherTargetRef ensures a Claude reference is not
// stripped when removing the Gemini reference and vice versa.
func TestRemoveRefDoesNotMatchOtherTargetRef(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	// File leads with the Claude reference; removing the Gemini ref must no-op.
	writeFile(t, "TARGET.md", claudeTarget.ref+"\n")

	modified, err := removeRef("TARGET.md", geminiTarget.ref, false)
	if err != nil {
		t.Fatalf("removeRef failed: %v", err)
	}
	if modified {
		t.Fatal("expected modified=false: gemini ref must not match claude ref")
	}
	if got := readFile(t, "TARGET.md"); got != claudeTarget.ref+"\n" {
		t.Fatalf("content modified unexpectedly: %q", got)
	}
}
