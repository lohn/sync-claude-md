package sync

import (
	"os"
	"strings"
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

// TestRemoveRefRemovesMovedReference strips the reference even when the user has
// moved it below other content, mirroring updateTarget's "present anywhere" rule.
func TestRemoveRefRemovesMovedReference(t *testing.T) {
	for _, rc := range refCases {
		t.Run(rc.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			chdir(t, tmpDir)

			writeFile(t, "TARGET.md", "# Title\n\n"+rc.ref+"\n")

			if !applyPlanCleanup(t, "TARGET.md", rc.ref) {
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

			if applyPlanCleanup(t, "TARGET.md", rc.ref) {
				t.Fatal("expected modified=false for substring-only reference")
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

// TestPlanSyncRejectsOversizedFile errors out, without writing, when an
// existing target file exceeds maxTargetFileSize.
func TestPlanSyncRejectsOversizedFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "TARGET.md", strings.Repeat("a", maxTargetFileSize+1))

	if _, err := planSync("TARGET.md", claudeTarget.ref); err == nil {
		t.Fatal("expected an error for an oversized target file")
	}
}

// TestPlanCleanupRejectsOversizedFile errors out, without writing, when an
// existing target file exceeds maxTargetFileSize.
func TestPlanCleanupRejectsOversizedFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "TARGET.md", strings.Repeat("a", maxTargetFileSize+1))

	if _, err := planCleanup("TARGET.md", claudeTarget.ref); err == nil {
		t.Fatal("expected an error for an oversized target file")
	}
}

// TestIsRefLineSkipsOverlongLines documents the line-length safeguard's
// trade-off: a line padded with whitespace far past maxLineLength is treated
// as a non-match even though it would trim down to exactly ref, since
// matching it would require paying for a full trim/compare on every
// pathologically long line.
func TestIsRefLineSkipsOverlongLines(t *testing.T) {
	padded := strings.Repeat(" ", maxLineLength) + claudeTarget.ref + strings.Repeat(" ", maxLineLength)

	if isRefLine(padded, claudeTarget.ref) {
		t.Fatal("expected a whitespace-padded line past maxLineLength to be skipped")
	}
	if !isRefLine(claudeTarget.ref, claudeTarget.ref) {
		t.Fatal("expected the bare reference line to match")
	}
}

// TestRemoveRefPreservesOverlongLine ensures a long non-reference line is
// preserved verbatim, and a normal reference line elsewhere in the same file
// is still removed.
func TestRemoveRefPreservesOverlongLine(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	longLine := strings.Repeat("x", maxLineLength*2)
	// A trailing blank line immediately after the reference is dropped by
	// withRefRemoved (see TestRemoveRefKeepsContent), so the long line is left
	// without its own trailing newline.
	writeFile(t, "TARGET.md", longLine+"\n"+claudeTarget.ref+"\n")

	if !applyPlanCleanup(t, "TARGET.md", claudeTarget.ref) {
		t.Fatal("expected modified=true")
	}
	if got := readFile(t, "TARGET.md"); got != longLine {
		t.Fatalf("long line not preserved: got len %d, want len %d", len(got), len(longLine))
	}
}
