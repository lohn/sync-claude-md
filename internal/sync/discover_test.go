package sync

import (
	"reflect"
	"sort"
	"testing"
)

// TestFilterAgentsFilesSplitsByExistence sorts existing files into toSync and
// missing files into deleted, ignoring non-AGENTS.md entries.
func TestFilterAgentsFilesSplitsByExistence(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "src/AGENTS.md", "# Agents\n")

	toSync, deleted, err := filterAgentsFiles([]string{
		"AGENTS.md",      // exists
		"src/AGENTS.md",  // exists
		"docs/AGENTS.md", // missing -> deleted
		"README.md",      // not AGENTS.md -> ignored
		"src/CLAUDE.md",  // not AGENTS.md -> ignored
	})
	if err != nil {
		t.Fatalf("filterAgentsFiles failed: %v", err)
	}

	wantSync := []string{"AGENTS.md", "src/AGENTS.md"}
	if !reflect.DeepEqual(toSync, wantSync) {
		t.Fatalf("toSync = %v, want %v", toSync, wantSync)
	}
	wantDeleted := []string{"docs/AGENTS.md"}
	if !reflect.DeepEqual(deleted, wantDeleted) {
		t.Fatalf("deleted = %v, want %v", deleted, wantDeleted)
	}
}

// TestFindAllAgentsSkipsExcludedDirs walks the tree and skips directories such
// as .git and node_modules.
func TestFindAllAgentsSkipsExcludedDirs(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "src/AGENTS.md", "# Agents\n")
	writeFile(t, ".git/AGENTS.md", "# Agents\n")
	writeFile(t, "node_modules/pkg/AGENTS.md", "# Agents\n")
	writeFile(t, ".hidden/AGENTS.md", "# Agents\n") // hidden but not excluded

	got, err := findAllAgents()
	if err != nil {
		t.Fatalf("findAllAgents failed: %v", err)
	}
	sort.Strings(got)

	want := []string{".hidden/AGENTS.md", "AGENTS.md", "src/AGENTS.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findAllAgents = %v, want %v", got, want)
	}
}

// TestFindTargetFilesMatchesSelectedTargets returns only the target files whose
// filename belongs to a selected target, paired with the right reference.
func TestFindTargetFilesMatchesSelectedTargets(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")
	writeFile(t, "GEMINI.md", "@./AGENTS.md\n")
	writeFile(t, "src/CLAUDE.md", "@AGENTS.md\n")
	writeFile(t, "README.md", "# readme\n") // unrelated

	// Only Gemini selected: CLAUDE.md files must be ignored.
	got, err := findTargetFiles([]target{geminiTarget})
	if err != nil {
		t.Fatalf("findTargetFiles failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target file, got %d: %+v", len(got), got)
	}
	if got[0].path != "GEMINI.md" || got[0].ref != geminiTarget.ref {
		t.Fatalf("unexpected target file: %+v", got[0])
	}

	// Both selected: all CLAUDE.md and GEMINI.md files, each with its own ref.
	got, err = findTargetFiles([]target{claudeTarget, geminiTarget})
	if err != nil {
		t.Fatalf("findTargetFiles failed: %v", err)
	}
	refByPath := make(map[string]string, len(got))
	for _, f := range got {
		refByPath[f.path] = f.ref
	}
	want := map[string]string{
		"CLAUDE.md":     claudeTarget.ref,
		"GEMINI.md":     geminiTarget.ref,
		"src/CLAUDE.md": claudeTarget.ref,
	}
	if !reflect.DeepEqual(refByPath, want) {
		t.Fatalf("findTargetFiles = %v, want %v", refByPath, want)
	}
}

// TestFindAgentsFilesPriorityExplicitFiles verifies explicit Files take
// priority over the --all scan.
func TestFindAgentsFilesPriorityExplicitFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "src/AGENTS.md", "# Agents\n")

	// Even with All set, explicit Files win.
	toSync, deleted, err := findAgentsFiles(Options{All: true, Files: []string{"src/AGENTS.md"}})
	if err != nil {
		t.Fatalf("findAgentsFiles failed: %v", err)
	}
	if !reflect.DeepEqual(toSync, []string{"src/AGENTS.md"}) {
		t.Fatalf("toSync = %v, want [src/AGENTS.md]", toSync)
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted = %v, want empty", deleted)
	}
}
