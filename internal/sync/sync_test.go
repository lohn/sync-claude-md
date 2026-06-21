package sync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory for testing.
func setupTestDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "sync-claude-md-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// chdir changes to the given directory and restores on cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// writeFile writes content to path relative to cwd.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// readFile reads content from path.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(b)
}

// TestCreateNewClaude creates CLAUDE.md when AGENTS.md exists but CLAUDE.md doesn't.
func TestCreateNewClaude(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "CLAUDE.md")
	if !strings.HasPrefix(content, "@AGENTS.md") {
		t.Fatalf("expected CLAUDE.md to start with @AGENTS.md, got:\n%s", content)
	}
}

// TestUpdateExistingClaude adds @AGENTS.md to an existing CLAUDE.md.
func TestUpdateExistingClaude(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# Existing Content\n\nSome text.\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "CLAUDE.md")
	lines := strings.Split(content, "\n")
	if strings.TrimSpace(lines[0]) != "@AGENTS.md" {
		t.Fatalf("expected first line to be @AGENTS.md, got:\n%s", content)
	}
	if !strings.Contains(content, "Existing Content") {
		t.Fatal("expected existing content to be preserved")
	}
}

// TestPrependInsertsBlankLine separates the inserted reference from existing
// content with a single blank line.
func TestPrependInsertsBlankLine(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# Existing\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "CLAUDE.md")
	if content != "@AGENTS.md\n\n# Existing\n" {
		t.Fatalf("expected a blank line after @AGENTS.md, got:\n%q", content)
	}
}

// TestReferenceAnywhereIsKept does not add a duplicate when @AGENTS.md already
// exists in CLAUDE.md, even if it is not the first line.
func TestReferenceAnywhereIsKept(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	original := "# Title\n\nSome intro.\n\n@AGENTS.md\n"
	writeFile(t, "CLAUDE.md", original)

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false when @AGENTS.md already present (not at top)")
	}

	if content := readFile(t, "CLAUDE.md"); content != original {
		t.Fatalf("content modified unexpectedly:\n%q", content)
	}
}

// TestIdempotent does nothing when @AGENTS.md is already present.
func TestIdempotent(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n\n# Existing\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for idempotent run")
	}

	content := readFile(t, "CLAUDE.md")
	if content != "@AGENTS.md\n\n# Existing\n" {
		t.Fatalf("content modified unexpectedly:\n%s", content)
	}
}

// TestCleanupRemovesReference removes @AGENTS.md when AGENTS.md is deleted.
func TestCleanupRemovesReference(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	// Setup: both files exist with reference
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n\n# Existing\n")

	// First run to sync
	_, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("setup Run failed: %v", err)
	}

	// Delete AGENTS.md
	_ = os.Remove("AGENTS.md")

	// Run again
	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true after cleanup")
	}

	content := readFile(t, "CLAUDE.md")
	if strings.Contains(content, "@AGENTS.md") {
		t.Fatalf("expected @AGENTS.md to be removed, got:\n%s", content)
	}
	if !strings.Contains(content, "Existing") {
		t.Fatal("expected existing content to be preserved")
	}
}

// TestCleanupDeletesEmptyFile deletes CLAUDE.md if it becomes empty.
func TestCleanupDeletesEmptyFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")

	// Sync first
	_, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("setup Run failed: %v", err)
	}

	// Delete AGENTS.md
	_ = os.Remove("AGENTS.md")

	// Run cleanup
	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to be deleted")
	}
}

// TestCheckMode returns error without making changes.
func TestCheckMode(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	// No CLAUDE.md

	changed, err := Run(Options{All: true, Check: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true in check mode")
	}

	// Verify no file was created
	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to NOT be created in check mode")
	}
}

// TestCheckModeNoChanges returns false when everything is up to date.
func TestCheckModeNoChanges(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")

	changed, err := Run(Options{All: true, Check: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false when up to date")
	}
}

// TestSubdirectory handles AGENTS.md in subdirectories.
func TestSubdirectory(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "src/AGENTS.md", "# Agents\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "src/CLAUDE.md")
	if !strings.HasPrefix(content, "@AGENTS.md") {
		t.Fatalf("expected src/CLAUDE.md to start with @AGENTS.md, got:\n%s", content)
	}
}

// TestMultipleDirectories handles AGENTS.md in multiple directories simultaneously.
func TestMultipleDirectories(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Root Agents\n")
	writeFile(t, "src/AGENTS.md", "# Src Agents\n")
	writeFile(t, "docs/AGENTS.md", "# Docs Agents\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	// Verify all CLAUDE.md files were created
	for _, path := range []string{"CLAUDE.md", "src/CLAUDE.md", "docs/CLAUDE.md"} {
		content := readFile(t, path)
		if !strings.HasPrefix(content, "@AGENTS.md") {
			t.Fatalf("expected %s to start with @AGENTS.md, got:\n%s", path, content)
		}
	}

	// Run again should be idempotent
	changed, err = Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false on second run")
	}
}

// TestExplicitFilesArgument processes only the provided files.
func TestExplicitFilesArgument(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Root Agents\n")
	writeFile(t, "src/AGENTS.md", "# Src Agents\n")

	// Only process root AGENTS.md
	changed, err := Run(Options{Files: []string{"AGENTS.md"}, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	// Root should have CLAUDE.md
	if _, err := os.Stat("CLAUDE.md"); os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to exist")
	}

	// src should NOT have CLAUDE.md yet
	if _, err := os.Stat("src/CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected src/CLAUDE.md to NOT exist")
	}
}

// TestCleanupMultipleDirectories removes references from multiple directories.
func TestCleanupMultipleDirectories(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Root Agents\n")
	writeFile(t, "src/AGENTS.md", "# Src Agents\n")
	writeFile(t, "docs/AGENTS.md", "# Docs Agents\n")

	// Initial sync
	_, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("setup Run failed: %v", err)
	}

	// Delete two AGENTS.md files
	_ = os.Remove("src/AGENTS.md")
	_ = os.Remove("docs/AGENTS.md")

	// Run cleanup
	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true after cleanup")
	}

	// Root should still have CLAUDE.md with reference
	content := readFile(t, "CLAUDE.md")
	if !strings.Contains(content, "@AGENTS.md") {
		t.Fatal("expected root CLAUDE.md to still have @AGENTS.md")
	}

	// src and docs CLAUDE.md should be deleted (empty after cleanup)
	if _, err := os.Stat("src/CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected src/CLAUDE.md to be deleted")
	}
	if _, err := os.Stat("docs/CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected docs/CLAUDE.md to be deleted")
	}
}

// TestSkipsGitDir verifies .git directory is skipped during scan.
func TestSkipsGitDir(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Root Agents\n")
	writeFile(t, ".git/AGENTS.md", "# Git Agents\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	// Root should have CLAUDE.md
	if _, err := os.Stat("CLAUDE.md"); os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to exist")
	}

	// .git should NOT have CLAUDE.md (skipped)
	if _, err := os.Stat(".git/CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected .git/CLAUDE.md to NOT exist")
	}
}

// TestHiddenDirIsScanned verifies hidden directories (except .git) are processed.
func TestHiddenDirIsScanned(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, ".hidden/AGENTS.md", "# Hidden Agents\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	// .hidden should have CLAUDE.md (not skipped)
	content := readFile(t, ".hidden/CLAUDE.md")
	if !strings.HasPrefix(content, "@AGENTS.md") {
		t.Fatalf("expected .hidden/CLAUDE.md to start with @AGENTS.md, got:\n%s", content)
	}
}

// TestGeminiCreate creates GEMINI.md with the @./AGENTS.md reference.
func TestGeminiCreate(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")

	changed, err := Run(Options{All: true, Gemini: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "GEMINI.md")
	if content != "@./AGENTS.md\n" {
		t.Fatalf("expected GEMINI.md to contain @./AGENTS.md, got:\n%q", content)
	}

	// CLAUDE.md must not be created when only --gemini is requested.
	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to NOT exist when only Gemini selected")
	}
}

// TestGeminiUpdatePrependsReference adds @./AGENTS.md to an existing GEMINI.md.
func TestGeminiUpdatePrependsReference(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "GEMINI.md", "# Existing\n")

	changed, err := Run(Options{All: true, Gemini: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	content := readFile(t, "GEMINI.md")
	if content != "@./AGENTS.md\n\n# Existing\n" {
		t.Fatalf("unexpected GEMINI.md content:\n%q", content)
	}
}

// TestSyncsBothTargets creates both CLAUDE.md and GEMINI.md when both targets
// are selected.
func TestSyncsBothTargets(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")

	changed, err := Run(Options{All: true, Claude: true, Gemini: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	if content := readFile(t, "CLAUDE.md"); content != "@AGENTS.md\n" {
		t.Fatalf("unexpected CLAUDE.md content:\n%q", content)
	}
	if content := readFile(t, "GEMINI.md"); content != "@./AGENTS.md\n" {
		t.Fatalf("unexpected GEMINI.md content:\n%q", content)
	}

	// Idempotent on a second run.
	changed, err = Run(Options{All: true, Claude: true, Gemini: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false on second run")
	}
}

// TestGeminiCleanupRemovesReference removes @./AGENTS.md and deletes the empty
// GEMINI.md when AGENTS.md is gone.
func TestGeminiCleanupRemovesReference(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")

	if _, err := Run(Options{All: true, Gemini: true}); err != nil {
		t.Fatalf("setup Run failed: %v", err)
	}

	_ = os.Remove("AGENTS.md")

	changed, err := Run(Options{All: true, Gemini: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true after cleanup")
	}

	if _, err := os.Stat("GEMINI.md"); !os.IsNotExist(err) {
		t.Fatal("expected GEMINI.md to be deleted")
	}
}

// TestCleanupOnlyTouchesSelectedTargets verifies that a cleanup scoped to one
// target leaves the other target's file untouched.
func TestCleanupOnlyTouchesSelectedTargets(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)

	writeFile(t, "AGENTS.md", "# Agents\n")

	// Create both targets.
	if _, err := Run(Options{All: true, Claude: true, Gemini: true}); err != nil {
		t.Fatalf("setup Run failed: %v", err)
	}

	_ = os.Remove("AGENTS.md")

	// Clean up only Gemini.
	if _, err := Run(Options{All: true, Gemini: true}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, err := os.Stat("GEMINI.md"); !os.IsNotExist(err) {
		t.Fatal("expected GEMINI.md to be deleted")
	}
	// CLAUDE.md should still carry its (now stale) reference since it was not selected.
	content := readFile(t, "CLAUDE.md")
	if !strings.Contains(content, "@AGENTS.md") {
		t.Fatalf("expected CLAUDE.md to be untouched, got:\n%q", content)
	}
}

// TestRunDestroyProtectionBlocksUnstagedTarget refuses to overwrite a target
// file that has unstaged changes, mirroring pre-commit's axisDestroy guard.
func TestRunDestroyProtectionBlocksUnstagedTarget(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n") // no reference yet
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")

	// Unstaged edit to CLAUDE.md; the sync wants to prepend the reference.
	writeFile(t, "CLAUDE.md", "# old\nwork in progress\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err == nil {
		t.Fatal("expected an error for unstaged CLAUDE.md")
	}
	var destroyErr *DestroyError
	if !errors.As(err, &destroyErr) {
		t.Fatalf("expected a *DestroyError, got: %v", err)
	}
	if len(destroyErr.Paths) != 1 || destroyErr.Paths[0] != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE.md in DestroyError.Paths, got %+v", destroyErr.Paths)
	}
	if changed {
		t.Fatal("expected changed=false when blocked")
	}
	if got := readFile(t, "CLAUDE.md"); got != "# old\nwork in progress\n" {
		t.Fatalf("CLAUDE.md was modified despite block: %q", got)
	}
}

// TestRunForceOverridesDestroyProtection allows Options.Force to write over a
// target with unstaged changes.
func TestRunForceOverridesDestroyProtection(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")
	writeFile(t, "CLAUDE.md", "# old\nwork in progress\n")

	changed, err := Run(Options{All: true, Claude: true, Force: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got := readFile(t, "CLAUDE.md"); !strings.HasPrefix(got, "@AGENTS.md") {
		t.Fatalf("expected @AGENTS.md to be prepended, got:\n%s", got)
	}
}

// TestRunDestroyProtectionSkippedOutsideGitRepo treats the absence of a git
// repository as "nothing to protect" rather than failing, since --all and
// explicit file lists do not otherwise require git.
func TestRunDestroyProtectionSkippedOutsideGitRepo(t *testing.T) {
	tmpDir := setupTestDir(t)
	chdir(t, tmpDir)
	// No git init.

	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n")

	changed, err := Run(Options{All: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got := readFile(t, "CLAUDE.md"); !strings.HasPrefix(got, "@AGENTS.md") {
		t.Fatalf("expected @AGENTS.md to be prepended, got:\n%s", got)
	}
}

// TestRunCheckModeIgnoresDestroyProtection reports drift even when the target
// has unstaged changes, since check mode never writes.
func TestRunCheckModeIgnoresDestroyProtection(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")
	writeFile(t, "CLAUDE.md", "# old\nwork in progress\n")

	changed, err := Run(Options{All: true, Check: true, Claude: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true in check mode")
	}
}
