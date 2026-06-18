// Package sync provides the core logic for synchronizing AGENTS.md to CLAUDE.md.
package sync

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// agentsRef is the import line a CLAUDE.md uses to reference its sibling
	// AGENTS.md.
	agentsRef = "@AGENTS.md"
	// claudeFileMode is the permission used when writing CLAUDE.md files.
	claudeFileMode os.FileMode = 0o644
)

// Options controls the behavior of Run.
type Options struct {
	All   bool     // scan entire repository
	Check bool     // dry-run, only validate
	Files []string // explicit file list (from pre-commit args)
}

// Run executes the synchronization.
// Returns true if any CLAUDE.md file was modified.
func Run(opts Options) (bool, error) {
	agentsFiles, deletedFiles, err := findAgentsFiles(opts)
	if err != nil {
		return false, err
	}

	changed := false

	// Sync: ensure CLAUDE.md references existing AGENTS.md
	for _, agentsPath := range agentsFiles {
		dir := filepath.Dir(agentsPath)
		claudePath := filepath.Join(dir, "CLAUDE.md")

		modified, err := syncFile(claudePath, opts.Check)
		if err != nil {
			return changed, fmt.Errorf("sync %s: %w", claudePath, err)
		}
		if modified {
			changed = true
		}
	}

	// Cleanup: remove references for deleted AGENTS.md
	for _, agentsPath := range deletedFiles {
		dir := filepath.Dir(agentsPath)
		claudePath := filepath.Join(dir, "CLAUDE.md")

		modified, err := removeAgentsRef(claudePath, opts.Check)
		if err != nil {
			return changed, fmt.Errorf("cleanup %s: %w", claudePath, err)
		}
		if modified {
			changed = true
		}
	}

	// Full cleanup: scan entire repo for stale references (only in --all mode)
	if opts.All {
		cleanupChanged, err := cleanupStaleClaude(agentsFiles, opts.Check)
		if err != nil {
			return changed, err
		}
		changed = changed || cleanupChanged
	}

	return changed, nil
}

// findAgentsFiles locates AGENTS.md files.
// Returns (filesToSync, filesDeleted, error).
// Priority: explicit Files > all scan > staged files
func findAgentsFiles(opts Options) ([]string, []string, error) {
	if len(opts.Files) > 0 {
		return filterAgentsFiles(opts.Files)
	}
	if opts.All {
		agents, err := findAllAgents()
		return agents, nil, err
	}
	return findStagedAgents()
}

// filterAgentsFiles extracts AGENTS.md paths from a list of files.
// Verifies file existence: existing files go to toSync, non-existing go to deleted.
func filterAgentsFiles(files []string) ([]string, []string, error) {
	var toSync []string
	var deleted []string
	for _, f := range files {
		if filepath.Base(f) != "AGENTS.md" {
			continue
		}
		if _, err := os.Stat(f); err == nil {
			toSync = append(toSync, f)
		} else if os.IsNotExist(err) {
			deleted = append(deleted, f)
		} else {
			return nil, nil, err
		}
	}
	return toSync, deleted, nil
}

// shouldSkipDir returns true if a directory should be skipped during walk.
func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "dist", "build":
		return true
	}
	return false
}

// findAllAgents recursively finds all AGENTS.md files.
func findAllAgents() ([]string, error) {
	var result []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "AGENTS.md" {
			result = append(result, path)
		}
		return nil
	})
	return result, err
}

// findStagedAgents finds AGENTS.md files in the git staged area.
// Returns (filesToSync, filesDeleted, error).
func findStagedAgents() ([]string, []string, error) {
	// Get added/modified/copied/renamed files
	cmd := execGit("diff", "--cached", "--name-only", "--diff-filter=ACMR")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff ACMR: %w", err)
	}

	var toSync []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if filepath.Base(line) == "AGENTS.md" {
			toSync = append(toSync, line)
		}
	}

	// Get deleted files
	cmd = execGit("diff", "--cached", "--name-only", "--diff-filter=D")
	out, err = cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff D: %w", err)
	}

	var deleted []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if filepath.Base(line) == "AGENTS.md" {
			deleted = append(deleted, line)
		}
	}

	return toSync, deleted, nil
}

// syncFile ensures CLAUDE.md exists with @AGENTS.md reference.
func syncFile(claudePath string, check bool) (bool, error) {
	exists := false
	if _, err := os.Stat(claudePath); err == nil {
		exists = true
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if !exists {
		if check {
			return true, nil
		}
		return true, createClaude(claudePath)
	}

	return updateClaude(claudePath, check)
}

// createClaude creates a new CLAUDE.md with @AGENTS.md reference.
func createClaude(claudePath string) error {
	return os.WriteFile(claudePath, []byte(agentsRef+"\n"), claudeFileMode)
}

// updateClaude ensures CLAUDE.md references @AGENTS.md. The reference may live
// anywhere in the file; only its absence triggers a write, in which case it is
// inserted at the top.
func updateClaude(claudePath string, check bool) (bool, error) {
	content, err := os.ReadFile(claudePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")

	// Already references @AGENTS.md somewhere? Then nothing to do.
	for _, line := range lines {
		if strings.TrimSpace(line) == agentsRef {
			return false, nil
		}
	}

	if check {
		return true, nil
	}

	// Insert @AGENTS.md at the top, dropping any leading blank lines so we do
	// not accumulate empty lines, and separating it from existing content with
	// a single blank line.
	firstNonEmpty := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = i
			break
		}
	}
	rest := lines[firstNonEmpty:]

	newLines := []string{agentsRef}
	if len(rest) > 0 {
		newLines = append(newLines, "")
	}
	newLines = append(newLines, rest...)

	newContent := strings.Join(newLines, "\n")
	return true, os.WriteFile(claudePath, []byte(newContent), claudeFileMode)
}

// cleanupStaleClaude removes @AGENTS.md references from CLAUDE.md files
// where AGENTS.md no longer exists.
func cleanupStaleClaude(agentsFiles []string, check bool) (bool, error) {
	// Build a set of directories that have AGENTS.md
	agentsDirs := make(map[string]bool)
	for _, path := range agentsFiles {
		agentsDirs[filepath.Dir(path)] = true
	}

	// Find all CLAUDE.md files
	var claudeFiles []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "CLAUDE.md" {
			claudeFiles = append(claudeFiles, path)
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	changed := false
	for _, claudePath := range claudeFiles {
		dir := filepath.Dir(claudePath)
		if agentsDirs[dir] {
			continue // AGENTS.md still exists, skip
		}

		modified, err := removeAgentsRef(claudePath, check)
		if err != nil {
			return changed, fmt.Errorf("cleanup %s: %w", claudePath, err)
		}
		if modified {
			changed = true
		}
	}

	return changed, nil
}

// removeAgentsRef removes the @AGENTS.md reference line from a CLAUDE.md file.
// Only removes the first occurrence at the top of the file (not inline references).
// Also removes immediately following blank lines to prevent empty line accumulation.
// If the file becomes empty after removal, it deletes the file.
func removeAgentsRef(claudePath string, check bool) (bool, error) {
	content, err := os.ReadFile(claudePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")

	// Find the first non-empty line
	firstNonEmpty := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = i
			break
		}
	}

	// If first non-empty line is not @AGENTS.md, skip
	if firstNonEmpty < 0 || strings.TrimSpace(lines[firstNonEmpty]) != agentsRef {
		return false, nil
	}

	if check {
		return true, nil
	}

	// Remove the @AGENTS.md line and any immediately following blank lines
	var newLines []string
	skipping := true // Start skipping after we pass the @AGENTS.md line
	for i, line := range lines {
		if i < firstNonEmpty {
			newLines = append(newLines, line)
			continue
		}
		if i == firstNonEmpty {
			// Skip the @AGENTS.md line itself
			continue
		}
		if skipping && strings.TrimSpace(line) == "" {
			// Skip blank lines immediately following @AGENTS.md
			continue
		}
		skipping = false
		newLines = append(newLines, line)
	}

	// Check if file is now empty (only whitespace left)
	hasContent := false
	for _, line := range newLines {
		if strings.TrimSpace(line) != "" {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return true, os.Remove(claudePath)
	}

	newContent := strings.Join(newLines, "\n")
	return true, os.WriteFile(claudePath, []byte(newContent), claudeFileMode)
}

// execGit runs a git command.
func execGit(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd
}
