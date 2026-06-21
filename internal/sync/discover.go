package sync

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// findAgentsFiles locates AGENTS.md files.
// Returns (filesToSync, filesDeleted, error).
// Priority: explicit Files > all scan > staged files. Outside a git
// repository "staged" is meaningless, so the default falls back to a full
// scan too (without the explicit --all flag, deletions go undetected, same as
// any non-all run that did not pick them up via Files or staged AGENTS.md).
func findAgentsFiles(opts Options) ([]string, []string, error) {
	if len(opts.Files) > 0 {
		return filterAgentsFiles(opts.Files)
	}
	if opts.All || !inGitRepo() {
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

// findAllAgents recursively finds all AGENTS.md files.
func findAllAgents() ([]string, error) {
	var result []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
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

// findTargetFiles walks the repository and returns every target file (CLAUDE.md
// or GEMINI.md) belonging to one of the given targets, paired with its
// reference line.
func findTargetFiles(targets []target) ([]targetFile, error) {
	// Map each target filename to its reference line so the walk can recognize
	// the files belonging to the selected targets.
	refByFilename := make(map[string]string, len(targets))
	for _, t := range targets {
		refByFilename[t.filename] = t.ref
	}

	var result []targetFile
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if ref, ok := refByFilename[d.Name()]; ok {
			result = append(result, targetFile{path: path, ref: ref})
		}
		return nil
	})
	return result, err
}

// execGit runs a git command.
func execGit(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	return cmd
}
