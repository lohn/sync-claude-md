package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory for testing and isolates the
// ambient git environment (see isolateGitEnv), since tests that never call
// initGitRepo rely on there being no git repository at all; without this, a
// test running nested inside another git operation (e.g. this suite running
// inside the pre-push hook) would inherit GIT_DIR and see the outer repo
// instead.
func setupTestDir(t *testing.T) string {
	t.Helper()
	isolateGitEnv(t)
	tmpDir, err := os.MkdirTemp("", "sync-claude-md-cmd-test-*")
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

// readFile reads content from path, failing the test if it is missing.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}

// fileExists reports whether path exists.
func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("failed to stat file %s: %v", path, err)
	return false
}

// isolateGitEnv removes git environment variables that would redirect git
// away from the current working directory, and pins config to empty files so
// global settings (hooks, signing) do not leak in. All changes are restored
// on cleanup.
func isolateGitEnv(t *testing.T) {
	t.Helper()
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
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
}

// initGitRepo creates a fresh git repository in a temp dir, chdirs into it,
// and configures a deterministic identity with signing disabled so commits
// never prompt. Returns the repo path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := setupTestDir(t)
	chdir(t, dir)
	runGit(t, "init", "-q")
	runGit(t, "config", "user.email", "test@example.com")
	runGit(t, "config", "user.name", "Test")
	runGit(t, "config", "commit.gpgsign", "false")
	runGit(t, "config", "core.hooksPath", os.DevNull)
	return dir
}

// runGit runs a git command in the current directory, failing the test on error.
func runGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// captureStderr redirects os.Stderr to a pipe for the duration of fn,
// restores it afterward, and returns whatever fn wrote. The pipe is drained
// by a goroutine running concurrently with fn, not after fn returns: fn
// writing more than the OS pipe buffer (~64KB) would otherwise deadlock,
// since nothing would be reading the other end yet.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		done <- result{out: buf.String(), err: err}
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	res := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close pipe reader: %v", err)
	}
	if res.err != nil {
		t.Fatalf("failed to read pipe: %v", res.err)
	}
	return res.out
}
