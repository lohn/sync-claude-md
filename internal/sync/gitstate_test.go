package sync

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// initGitRepo creates a fresh git repository in a temp dir, chdirs into it, and
// configures a deterministic identity with signing disabled so commits never
// prompt. Returns the repo path.
//
// It first scrubs the ambient git environment so the suite is hermetic: when run
// from inside another git operation (e.g. the pre-push hook), inherited GIT_DIR
// / GIT_INDEX_FILE and a global core.hooksPath would otherwise make git operate
// on the outer repo and fire its hooks on these temp commits.
func initGitRepo(t *testing.T) string {
	t.Helper()
	isolateGitEnv(t)

	dir := setupTestDir(t)
	chdir(t, dir)
	runGit(t, "init", "-q")
	runGit(t, "config", "user.email", "test@example.com")
	runGit(t, "config", "user.name", "Test")
	runGit(t, "config", "commit.gpgsign", "false")
	// Make sure no inherited hooks path fires on commits in this temp repo.
	runGit(t, "config", "core.hooksPath", os.DevNull)
	return dir
}

// isolateGitEnv removes git environment variables that would redirect git away
// from the current working directory, and pins config to empty files so global
// settings (hooks, signing) do not leak in. All changes are restored on cleanup.
func isolateGitEnv(t *testing.T) {
	t.Helper()
	// Unset variables that would override cwd-based repo discovery.
	for _, v := range []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY", "GIT_COMMON_DIR", "GIT_CONFIG",
	} {
		if orig, ok := os.LookupEnv(v); ok {
			name := v
			_ = os.Unsetenv(name)
			t.Cleanup(func() { _ = os.Setenv(name, orig) })
		}
	}
	// Pin config to empty files: no global hooksPath / gpgsign / identity leaks.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
}

// runGit runs a git command in the current directory, failing the test on error.
func runGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// claudeOpts builds Options for a plain CLAUDE.md sync run (staged AGENTS.md,
// the default git-hook discovery mode).
func claudeOpts() Options {
	return Options{Claude: true}
}

// TestSyncMissingRefFails covers axis B: a generated CLAUDE.md that is
// not staged must fail (the original bug — second run without re-staging).
func TestSyncMissingRefFails(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	// First run: creates CLAUDE.md, not staged -> sync violation.
	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected the run to write CLAUDE.md")
	}
	if len(res.SyncPaths) != 1 || res.SyncPaths[0] != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE.md sync violation, got %+v", res.SyncPaths)
	}

	// Second run without staging: still a violation (regression guard).
	res, err = Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run (2nd): %v", err)
	}
	if len(res.SyncPaths) != 1 {
		t.Fatalf("expected CLAUDE.md still unstaged on 2nd run, got %+v", res.SyncPaths)
	}
}

// TestSyncStagedPasses covers axis B cleared by a manual git add.
func TestSyncStagedPasses(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	if _, err := Run(claudeOpts()); err != nil {
		t.Fatalf("setup run: %v", err)
	}
	runGit(t, "add", "CLAUDE.md")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.SyncPaths) != 0 || len(res.DestroyPaths) != 0 {
		t.Fatalf("expected no violations, got sync=%+v destroy=%+v", res.SyncPaths, res.DestroyPaths)
	}
}

// TestStageAutoStages covers --stage: a one-pass run that stages the
// generated file and reports no remaining sync violation.
func TestStageAutoStages(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	opts := claudeOpts()
	opts.Stage = true
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Staged {
		t.Fatal("expected Staged=true")
	}
	if len(res.SyncPaths) != 0 {
		t.Fatalf("expected no sync violations after --stage, got %+v", res.SyncPaths)
	}
	// The reference must now be in the index.
	has, err := indexHasRef("CLAUDE.md", claudeTarget.ref)
	if err != nil {
		t.Fatalf("indexHasRef: %v", err)
	}
	if !has {
		t.Fatal("expected CLAUDE.md @AGENTS.md to be staged after --stage")
	}
}

// TestCleanupUnstagedFails covers axis B on deletion: removing the
// reference must be staged.
func TestCleanupUnstagedFails(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")

	// Stage the deletion of AGENTS.md.
	runGit(t, "rm", "-q", "AGENTS.md")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// CLAUDE.md becomes empty and is deleted on disk, but the index still has
	// the reference until staged -> sync violation.
	if len(res.SyncPaths) != 1 || res.SyncPaths[0] != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE.md sync violation, got %+v", res.SyncPaths)
	}

	// Stage the removal and re-run: clean.
	runGit(t, "add", "CLAUDE.md")
	res, err = Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run (2nd): %v", err)
	}
	if len(res.SyncPaths) != 0 {
		t.Fatalf("expected no violations after staging removal, got %+v", res.SyncPaths)
	}
}

// TestSyncDestroyProtection covers axis A: an update to a target with
// unstaged changes is refused without --force and nothing is written.
func TestSyncDestroyProtection(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n") // no reference yet
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")

	// Re-stage AGENTS.md so the staged-files discovery sees it.
	writeFile(t, "AGENTS.md", "# Agents v2\n")
	runGit(t, "add", "AGENTS.md")
	// Unstaged edit to CLAUDE.md; the sync wants to prepend the reference.
	writeFile(t, "CLAUDE.md", "# old\nwork in progress\n")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.DestroyPaths) != 1 || res.DestroyPaths[0] != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE.md destroy violation, got %+v", res.DestroyPaths)
	}
	if res.Wrote {
		t.Fatal("expected no write when destroy protection blocks")
	}
	if got := readFile(t, "CLAUDE.md"); got != "# old\nwork in progress\n" {
		t.Fatalf("CLAUDE.md was modified despite block: %q", got)
	}
}

// TestSyncForceOverridesDestroy covers --force: it writes over unstaged
// changes (the sync violation remains because it is not staged).
func TestSyncForceOverridesDestroy(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "# old\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")
	writeFile(t, "AGENTS.md", "# Agents v2\n")
	runGit(t, "add", "AGENTS.md")
	writeFile(t, "CLAUDE.md", "# old\nwork in progress\n")

	opts := claudeOpts()
	opts.Force = true
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected --force to write")
	}
	if len(res.DestroyPaths) != 0 {
		t.Fatalf("expected no destroy block with --force, got %+v", res.DestroyPaths)
	}
	if got := readFile(t, "CLAUDE.md"); got == "# old\nwork in progress\n" {
		t.Fatal("expected CLAUDE.md to be rewritten with --force")
	}
}

// TestUnrelatedUnstagedEditPasses covers axis A scoping: an unrelated
// unstaged edit does not trigger an error when no sync write is needed.
func TestUnrelatedUnstagedEditPasses(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n\n# notes\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")

	// Re-stage AGENTS.md so the discovery evaluates this directory.
	writeFile(t, "AGENTS.md", "# Agents v2\n")
	runGit(t, "add", "AGENTS.md")
	// Reference is already staged; an unrelated unstaged edit must not block.
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n\n# notes\nmore unstaged text\n")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.DestroyPaths) != 0 || len(res.SyncPaths) != 0 {
		t.Fatalf("expected no violations, got destroy=%+v sync=%+v", res.DestroyPaths, res.SyncPaths)
	}
}

// TestIgnoredTargetIsNoOp covers ignore handling, on by default: an ignored
// target is never created, verified, or reported.
func TestIgnoredTargetIsNoOp(t *testing.T) {
	initGitRepo(t)
	writeFile(t, ".gitignore", "CLAUDE.md\n")
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", ".gitignore", "AGENTS.md")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.SyncPaths) != 0 || len(res.DestroyPaths) != 0 {
		t.Fatalf("expected no violations for ignored target, got sync=%+v destroy=%+v", res.SyncPaths, res.DestroyPaths)
	}
	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to NOT be created when ignored")
	}
}

// TestNoIgnoreProcessesIgnoredTarget covers --no-ignore: it overrides the
// default ignore-skip and processes the target anyway.
func TestNoIgnoreProcessesIgnoredTarget(t *testing.T) {
	initGitRepo(t)
	writeFile(t, ".gitignore", "CLAUDE.md\n")
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", ".gitignore", "AGENTS.md")

	opts := claudeOpts()
	opts.NoIgnore = true
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected the run to write the ignored CLAUDE.md")
	}
	if content := readFile(t, "CLAUDE.md"); !strings.HasPrefix(content, "@AGENTS.md") {
		t.Fatalf("expected CLAUDE.md to start with @AGENTS.md, got:\n%s", content)
	}
}

// TestGeminiIndexSync covers the GEMINI.md target through both axes in brief.
func TestGeminiIndexSync(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	opts := Options{Gemini: true}

	// Axis B: generated GEMINI.md not staged.
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.SyncPaths) != 1 || res.SyncPaths[0] != "GEMINI.md" {
		t.Fatalf("expected GEMINI.md sync violation, got %+v", res.SyncPaths)
	}

	// Stage clears it.
	runGit(t, "add", "GEMINI.md")
	res, err = Run(opts)
	if err != nil {
		t.Fatalf("Run (2nd): %v", err)
	}
	if len(res.SyncPaths) != 0 {
		t.Fatalf("expected no violations after staging GEMINI.md, got %+v", res.SyncPaths)
	}
}

// TestWroteFalseOnNoOp verifies res.Wrote reflects whether anything was
// actually written: a fully-synced, staged tree is a no-op.
func TestWroteFalseOnNoOp(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")

	res, err := Run(claudeOpts())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Wrote {
		t.Fatal("expected Wrote=false when everything is already synced")
	}
	if len(res.SyncPaths) != 0 || len(res.DestroyPaths) != 0 {
		t.Fatalf("expected no violations, got sync=%+v destroy=%+v", res.SyncPaths, res.DestroyPaths)
	}
}

// TestStageDeletesUntrackedTarget covers the --stage path when cleanup
// removes an untracked target file: git add must not fail on the now-absent,
// never-tracked path (the isStageable filter in gitAdd).
func TestStageDeletesUntrackedTarget(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")
	runGit(t, "commit", "-qm", "init")

	// Stage the deletion of AGENTS.md so the sync plans a cleanup.
	runGit(t, "rm", "-q", "AGENTS.md")
	// An untracked CLAUDE.md whose only content is the reference: cleanup deletes
	// it entirely, leaving a path that is neither on disk nor tracked.
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")

	opts := claudeOpts()
	opts.Force = true // bypass destroy protection on the untracked file
	opts.Stage = true
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Staged {
		t.Fatal("expected Staged=true")
	}
	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to be deleted by cleanup")
	}
}

// TestCheckDetectsUnstagedSync covers check's index-sync axis: a CLAUDE.md
// already correct on disk but never staged must still be reported as drift
// (the same "second run silently passes" bug, now also caught without
// writing).
func TestCheckDetectsUnstagedSync(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")
	runGit(t, "add", "AGENTS.md") // CLAUDE.md deliberately left untracked

	opts := claudeOpts()
	opts.Check = true
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Changed {
		t.Fatal("expected Changed=true for an unstaged-but-correct CLAUDE.md")
	}

	runGit(t, "add", "CLAUDE.md")
	res, err = Run(opts)
	if err != nil {
		t.Fatalf("Run (2nd): %v", err)
	}
	if res.Changed {
		t.Fatal("expected Changed=false once CLAUDE.md is staged")
	}
}

// TestAllOrphanCleanupReportsSyncViolation covers the --all sweep combined
// with index-sync: a committed CLAUDE.md whose AGENTS.md no longer exists
// anywhere is deleted on disk, but the deletion still needs staging.
func TestAllOrphanCleanupReportsSyncViolation(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")
	runGit(t, "add", "CLAUDE.md")
	runGit(t, "commit", "-qm", "init")

	opts := Options{All: true, Claude: true}
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat("CLAUDE.md"); !os.IsNotExist(err) {
		t.Fatal("expected orphan CLAUDE.md to be deleted")
	}
	if len(res.SyncPaths) != 1 || res.SyncPaths[0] != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE.md sync violation for the unstaged deletion, got %+v", res.SyncPaths)
	}

	runGit(t, "add", "CLAUDE.md")
	res, err = Run(opts)
	if err != nil {
		t.Fatalf("Run (2nd): %v", err)
	}
	if len(res.SyncPaths) != 0 {
		t.Fatalf("expected no violations after staging the deletion, got %+v", res.SyncPaths)
	}
}
