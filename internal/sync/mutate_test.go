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

// applyPlanSync plans and applies a sync mutation, returning whether it modified
// anything — a small helper mirroring the old syncFile signature for the tests.
func applyPlanSync(t *testing.T, path, ref string) bool {
	t.Helper()
	a, err := planSync(path, ref)
	if err != nil {
		t.Fatalf("planSync failed: %v", err)
	}
	if err := applyAction(a); err != nil {
		t.Fatalf("applyAction failed: %v", err)
	}
	return a.modifies()
}

// applyPlanCleanup plans and applies a cleanup mutation, returning whether it
// modified anything.
func applyPlanCleanup(t *testing.T, path, ref string) bool {
	t.Helper()
	a, err := planCleanup(path, ref)
	if err != nil {
		t.Fatalf("planCleanup failed: %v", err)
	}
	if err := applyAction(a); err != nil {
		t.Fatalf("applyAction failed: %v", err)
	}
	return a.modifies()
}

// TestSyncFileCreates writes a fresh target file containing only the reference.
func TestSyncFileCreates(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			if !applyPlanSync(t, "TARGET.md", rc.ref) {
				t.Fatal("expected modified=true")
			}
			if got := readFile(t, "TARGET.md"); got != rc.ref+"\n" {
				t.Fatalf("got %q, want %q", got, rc.ref+"\n")
			}
		})
	}
}

// TestPlanSyncCheckModeDoesNotCreate reports drift without writing the file.
func TestPlanSyncCheckModeDoesNotCreate(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			a, err := planSync("TARGET.md", rc.ref)
			if err != nil {
				t.Fatalf("planSync failed: %v", err)
			}
			if !a.modifies() {
				t.Fatal("expected planned action to modify")
			}
			// Planning alone must not touch disk.
			if _, err := os.Stat("TARGET.md"); !os.IsNotExist(err) {
				t.Fatal("expected TARGET.md to NOT be created by planning")
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

			if !applyPlanSync(t, "TARGET.md", rc.ref) {
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

			if applyPlanSync(t, "TARGET.md", rc.ref) {
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

			if !applyPlanCleanup(t, "TARGET.md", rc.ref) {
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

			if !applyPlanCleanup(t, "TARGET.md", rc.ref) {
				t.Fatal("expected modified=true")
			}
			if _, err := os.Stat("TARGET.md"); !os.IsNotExist(err) {
				t.Fatal("expected TARGET.md to be deleted")
			}
		})
	}
}

// TestRemoveRefSkipsInlineReference does nothing when the reference is not the
// first non-empty line.
func TestRemoveRefSkipsInlineReference(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			original := "# Title\n\n" + rc.ref + "\n"
			writeFile(t, "TARGET.md", original)

			if applyPlanCleanup(t, "TARGET.md", rc.ref) {
				t.Fatal("expected modified=false for inline reference")
			}
			if got := readFile(t, "TARGET.md"); got != original {
				t.Fatalf("content modified unexpectedly: %q", got)
			}
		})
	}
}

// TestRemoveRefMissingFileNoOps returns a no-op action when the target file does
// not exist, so cleanup of a deleted AGENTS.md never fails for a target that
// was never synced in that directory.
func TestRemoveRefMissingFileNoOps(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			if applyPlanCleanup(t, "MISSING.md", rc.ref) {
				t.Fatal("expected modified=false for missing file")
			}
		})
	}
}

// TestPlanCleanupCheckModeDoesNotWrite ensures planning a removal does not touch
// the file.
func TestPlanCleanupCheckModeDoesNotWrite(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			original := rc.ref + "\n"
			writeFile(t, "TARGET.md", original)

			a, err := planCleanup("TARGET.md", rc.ref)
			if err != nil {
				t.Fatalf("planCleanup failed: %v", err)
			}
			if !a.modifies() {
				t.Fatal("expected planned action to modify")
			}
			if got := readFile(t, "TARGET.md"); got != original {
				t.Fatalf("file changed by planning: %q", got)
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

	if applyPlanCleanup(t, "TARGET.md", geminiTarget.ref) {
		t.Fatal("expected modified=false: gemini ref must not match claude ref")
	}
	if got := readFile(t, "TARGET.md"); got != claudeTarget.ref+"\n" {
		t.Fatalf("content modified unexpectedly: %q", got)
	}
}
