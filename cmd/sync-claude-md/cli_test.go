package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binPath is the freshly built sync-claude-md binary the tests in this file
// exercise as a real subprocess. main()'s "sync"/"check"/unknown-command
// branches call os.Exit directly, so they cannot be invoked in-process
// without killing the test binary; black-box execution is the only way to
// cover them.
var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "sync-claude-md-cli-test-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create temp dir:", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	binPath = filepath.Join(tmpDir, "sync-claude-md")
	// -buildvcs=false: building from a worktree whose checkout lives under
	// the parent repo's tree fails VCS stamping (see ../../AGENTS.md).
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runBinary runs the built sync-claude-md binary with args in dir, returning
// stdout, stderr, and the process exit code.
func runBinary(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("failed to run binary: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// TestCLINoArgsPrintsHelp covers main()'s len(os.Args) < 2 branch.
func TestCLINoArgsPrintsHelp(t *testing.T) {
	stdout, stderr, code := runBinary(t, t.TempDir())
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
	}
	if stdout != helpText {
		t.Errorf("stdout = %q, want helpText", stdout)
	}
}

// TestCLIHelpFlags covers every spelling of the help flag main() recognizes.
func TestCLIHelpFlags(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		t.Run(arg, func(t *testing.T) {
			stdout, stderr, code := runBinary(t, t.TempDir(), arg)
			if code != 0 {
				t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
			}
			if stdout != helpText {
				t.Errorf("stdout = %q, want helpText", stdout)
			}
		})
	}
}

// TestCLIVersionFlag covers main()'s --version branch. The exact
// version/commit/date values depend on how this test binary itself was
// built, so this only checks the format, not literal values.
func TestCLIVersionFlag(t *testing.T) {
	stdout, stderr, code := runBinary(t, t.TempDir(), "--version")
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr: %s", code, stderr)
	}
	if !strings.HasPrefix(stdout, "sync-claude-md ") || !strings.Contains(stdout, "(commit: ") || !strings.Contains(stdout, "built: ") {
		t.Errorf("stdout = %q, want %q", stdout, "sync-claude-md <version> (commit: <commit>, built: <date>)\n")
	}
}

// TestCLIUnknownCommand covers main()'s default branch: an unrecognized
// first argument exits 1 with an error plus the help text, on stderr.
func TestCLIUnknownCommand(t *testing.T) {
	stdout, stderr, code := runBinary(t, t.TempDir(), "bogus")
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, `unknown command "bogus"`) || !strings.Contains(stderr, helpText) {
		t.Errorf("stderr = %q, want the unknown-command error followed by helpText", stderr)
	}
}

// TestCLISyncAndCheckDispatch exercises main()'s "sync" and "check" branches
// end to end: real argument passing through os.Args[2:] and the os.Exit call
// on each subcommand's returned code, complementing the in-process
// runSync/runCheck tests in main_test.go.
func TestCLISyncAndCheckDispatch(t *testing.T) {
	isolateGitEnv(t)
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	_, stderr, code := runBinary(t, dir, "sync", "--force")
	if code != 0 {
		t.Fatalf("sync --force: code = %d, want 0, stderr: %s", code, stderr)
	}
	claudePath := filepath.Join(dir, "CLAUDE.md")
	got, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	if string(got) != "@AGENTS.md\n" {
		t.Fatalf("CLAUDE.md = %q, want %q", got, "@AGENTS.md\n")
	}

	_, stderr, code = runBinary(t, dir, "check")
	if code != 0 {
		t.Fatalf("check: code = %d, want 0, stderr: %s", code, stderr)
	}
}
