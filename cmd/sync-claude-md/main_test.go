package main

import (
	"flag"
	"runtime/debug"
	"strings"
	"testing"
)

// TestVersionFromBuildInfo covers the two independent sources build info can
// supply — module version and VCS stamps — including the cases where one or
// both are absent (e.g. `go install pkg@latest`, which resolves a module
// version but has no VCS checkout to stamp from).
func TestVersionFromBuildInfo(t *testing.T) {
	cases := []struct {
		name        string
		info        *debug.BuildInfo
		wantVersion string
		wantCommit  string
		wantDate    string
	}{
		{
			name: "module version and vcs stamps present",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
				},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "abc123",
			wantDate:    "2024-01-01T00:00:00Z",
		},
		{
			name: "module version only, no vcs stamps (go install pkg@latest)",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.3"},
			},
			wantVersion: "v1.2.3",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name: "devel version is not a usable replacement",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
			},
			wantVersion: "dev",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name:        "empty build info changes nothing",
			info:        &debug.BuildInfo{},
			wantVersion: "dev",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVersion, gotCommit, gotDate := versionFromBuildInfo(tc.info, "dev", "none", "unknown")
			if gotVersion != tc.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tc.wantVersion)
			}
			if gotCommit != tc.wantCommit {
				t.Errorf("commit = %q, want %q", gotCommit, tc.wantCommit)
			}
			if gotDate != tc.wantDate {
				t.Errorf("date = %q, want %q", gotDate, tc.wantDate)
			}
		})
	}
}

// TestSelectTargets covers all four combinations of --no-claude/--gemini:
// CLAUDE.md is on by default, --gemini adds GEMINI.md, --no-claude opts
// CLAUDE.md out, and --no-claude without --gemini leaves nothing selected.
func TestSelectTargets(t *testing.T) {
	cases := []struct {
		name       string
		noClaude   bool
		gemini     bool
		wantClaude bool
		wantGemini bool
		wantOK     bool
	}{
		{name: "default: claude only", noClaude: false, gemini: false, wantClaude: true, wantGemini: false, wantOK: true},
		{name: "no-claude without gemini: nothing selected", noClaude: true, gemini: false, wantClaude: false, wantGemini: false, wantOK: false},
		{name: "no-claude with gemini: gemini only", noClaude: true, gemini: true, wantClaude: false, wantGemini: true, wantOK: true},
		{name: "gemini added alongside claude", noClaude: false, gemini: true, wantClaude: true, wantGemini: true, wantOK: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var claude, gemini, ok bool
			stderr := captureStderr(t, func() {
				claude, gemini, ok = selectTargets(tc.noClaude, tc.gemini)
			})
			if claude != tc.wantClaude || gemini != tc.wantGemini || ok != tc.wantOK {
				t.Fatalf("selectTargets(%v, %v) = (%v, %v, %v), want (%v, %v, %v)",
					tc.noClaude, tc.gemini, claude, gemini, ok, tc.wantClaude, tc.wantGemini, tc.wantOK)
			}
			if !tc.wantOK && !strings.Contains(stderr, "--no-claude without --gemini") {
				t.Errorf("stderr = %q, want it to explain the --no-claude/--gemini conflict", stderr)
			}
		})
	}
}

// TestPrintFlags checks the alignment and aliasing rules printFlags promises:
// single-letter shorthand flags are never shown on their own line, and a
// shorthand listed in the aliases map is appended to its long form instead.
func TestPrintFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var force, stage bool
	fs.BoolVar(&force, "force", false, "Overwrite anyway")
	fs.BoolVar(&force, "f", false, "Shorthand for --force")
	fs.BoolVar(&stage, "stage", false, "Stage the result")

	out := captureStderr(t, func() {
		printFlags(fs, map[string]string{"force": "-f"})
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (the lone shorthand -f must not get its own line):\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "--force, -f") || !strings.Contains(lines[0], "Overwrite anyway") {
		t.Errorf("line 0 = %q, want it to show --force aliased to -f", lines[0])
	}
	if !strings.Contains(lines[1], "--stage") || !strings.Contains(lines[1], "Stage the result") {
		t.Errorf("line 1 = %q, want it to show --stage with no alias", lines[1])
	}
	if strings.Contains(lines[1], "-f") {
		t.Errorf("line 1 = %q, --stage has no alias and should not mention -f", lines[1])
	}
}

// TestRunSyncOutsideGitRequiresForce covers Result.NoGitPaths: outside a git
// repository, sync refuses to write anything at all -- even a brand-new file
// -- without --force, since there is no git history to recover from.
func TestRunSyncOutsideGitRequiresForce(t *testing.T) {
	chdir(t, setupTestDir(t))
	writeFile(t, "AGENTS.md", "# Agents\n")

	var code int
	stderr := captureStderr(t, func() { code = runSync(nil) })
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "refusing to write outside a git repository") || !strings.Contains(stderr, "--force") {
		t.Errorf("stderr = %q, want the no-git refusal with a --force hint", stderr)
	}
	if fileExists(t, "CLAUDE.md") {
		t.Error("CLAUDE.md should not have been written")
	}

	stderr = captureStderr(t, func() { code = runSync([]string{"--force"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
	}
	if got := readFile(t, "CLAUDE.md"); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q, want %q", got, "@AGENTS.md\n")
	}
}

// TestRunSyncIndexSyncViolationRequiresStaging covers Result.SyncPaths via the
// default (no --all) staged-AGENTS.md discovery path: a freshly created
// CLAUDE.md that is not yet staged would miss the next commit, so it must be
// reported until staged (--stage, or a manual git add).
func TestRunSyncIndexSyncViolationRequiresStaging(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	var code int
	stderr := captureStderr(t, func() { code = runSync(nil) })
	if code != 1 {
		t.Fatalf("code = %d, want 1, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "git add -- CLAUDE.md") {
		t.Errorf("stderr = %q, want a git add hint for CLAUDE.md", stderr)
	}
	if got := readFile(t, "CLAUDE.md"); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q, want %q", got, "@AGENTS.md\n")
	}

	// Re-running without staging is still a violation (the original bug this
	// guarantee exists for: a second run must not silently look clean).
	stderr = captureStderr(t, func() { code = runSync(nil) })
	if code != 1 {
		t.Fatalf("second run: code = %d, want 1, stderr: %s", code, stderr)
	}

	stderr = captureStderr(t, func() { code = runSync([]string{"--stage"}) })
	if code != 0 {
		t.Fatalf("with --stage: code = %d, want 0, stderr: %s", code, stderr)
	}
}

// TestRunSyncDestroyProtection covers Result.DestroyPaths: an existing target
// file with unstaged changes must not be silently overwritten, since that
// would discard work that was never committed. --force (together with
// --stage, to also clear the resulting index-sync violation) lifts the block.
func TestRunSyncDestroyProtection(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "stale\n")
	runGit(t, "add", "AGENTS.md", "CLAUDE.md")
	runGit(t, "commit", "-q", "-m", "initial")

	// Unstaged edit: planSync wants to prepend the reference, which would
	// clobber this.
	writeFile(t, "CLAUDE.md", "stale\nedited\n")

	var code int
	stderr := captureStderr(t, func() { code = runSync([]string{"--all"}) })
	if code != 1 {
		t.Fatalf("code = %d, want 1, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "refusing to overwrite files with unstaged changes") || !strings.Contains(stderr, "CLAUDE.md") {
		t.Errorf("stderr = %q, want the destroy-protection refusal naming CLAUDE.md", stderr)
	}
	if got := readFile(t, "CLAUDE.md"); got != "stale\nedited\n" {
		t.Errorf("CLAUDE.md = %q, want it untouched", got)
	}

	stderr = captureStderr(t, func() { code = runSync([]string{"--all", "--force", "--stage"}) })
	if code != 0 {
		t.Fatalf("with --force --stage: code = %d, want 0, stderr: %s", code, stderr)
	}
	want := "@AGENTS.md\n\nstale\nedited\n"
	if got := readFile(t, "CLAUDE.md"); got != want {
		t.Errorf("CLAUDE.md = %q, want %q", got, want)
	}
}

// TestRunSyncFailOnChangeBlocksOtherwiseCleanRun covers --fail-on-change:
// it runs last and never blocks the write or stage, it only flips an
// otherwise-successful exit code to 1 because something was written.
func TestRunSyncFailOnChangeBlocksOtherwiseCleanRun(t *testing.T) {
	initGitRepo(t)
	writeFile(t, "AGENTS.md", "# Agents\n")
	runGit(t, "add", "AGENTS.md")

	var code int
	stderr := captureStderr(t, func() { code = runSync([]string{"--stage", "--fail-on-change"}) })
	if code != 1 {
		t.Fatalf("code = %d, want 1, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "agent instruction files were updated") {
		t.Errorf("stderr = %q, want the fail-on-change message", stderr)
	}
	// The write and stage themselves must still have gone through.
	if got := readFile(t, "CLAUDE.md"); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q, want %q", got, "@AGENTS.md\n")
	}
}

// TestRunSyncNoClaudeWithoutGeminiErrors covers the CLI-only validation that
// rejects --no-claude without --gemini before any file is touched.
func TestRunSyncNoClaudeWithoutGeminiErrors(t *testing.T) {
	chdir(t, setupTestDir(t))

	var code int
	stderr := captureStderr(t, func() { code = runSync([]string{"--no-claude"}) })
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "--no-claude without --gemini") {
		t.Errorf("stderr = %q, want the no-claude/gemini conflict message", stderr)
	}
}

// TestRunSyncGeminiOnlyWiresFlagsThroughOptions confirms --no-claude --gemini
// reaches sync.Options correctly: only GEMINI.md is written, never CLAUDE.md.
func TestRunSyncGeminiOnlyWiresFlagsThroughOptions(t *testing.T) {
	chdir(t, setupTestDir(t))
	writeFile(t, "AGENTS.md", "# Agents\n")

	var code int
	stderr := captureStderr(t, func() { code = runSync([]string{"--force", "--no-claude", "--gemini"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
	}
	if got := readFile(t, "GEMINI.md"); got != "@./AGENTS.md\n" {
		t.Errorf("GEMINI.md = %q, want %q", got, "@./AGENTS.md\n")
	}
	if fileExists(t, "CLAUDE.md") {
		t.Error("CLAUDE.md should not have been written with --no-claude")
	}
}

// TestRunCheckCleanIsExitZero covers Result.Changed == false: a target
// already in sync on disk reports no drift and writes nothing.
func TestRunCheckCleanIsExitZero(t *testing.T) {
	chdir(t, setupTestDir(t))
	writeFile(t, "AGENTS.md", "# Agents\n")
	writeFile(t, "CLAUDE.md", "@AGENTS.md\n")

	var code int
	stderr := captureStderr(t, func() { code = runCheck(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
	}
}

// TestRunCheckDriftReportsAndNeverWrites covers Result.Changed == true: a
// missing CLAUDE.md is reported as drift, but check must never write it.
func TestRunCheckDriftReportsAndNeverWrites(t *testing.T) {
	chdir(t, setupTestDir(t))
	writeFile(t, "AGENTS.md", "# Agents\n")

	var code int
	stderr := captureStderr(t, func() { code = runCheck(nil) })
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "agent instruction files are out of sync") {
		t.Errorf("stderr = %q, want the drift message", stderr)
	}
	if fileExists(t, "CLAUDE.md") {
		t.Error("check must never write CLAUDE.md")
	}
}
