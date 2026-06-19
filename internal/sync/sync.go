// Package sync provides the core logic for synchronizing AGENTS.md to the
// per-agent instruction files CLAUDE.md and GEMINI.md.
package sync

import (
	"fmt"
	"path/filepath"
)

// Options controls the behavior of Run.
type Options struct {
	All    bool     // scan entire repository
	Check  bool     // dry-run, only validate
	Files  []string // explicit file list (from pre-commit args)
	Claude bool     // sync CLAUDE.md
	Gemini bool     // sync GEMINI.md
}

// Run executes the synchronization.
// Returns true if any target file was modified.
func Run(opts Options) (bool, error) {
	targets := resolveTargets(opts)

	agentsFiles, deletedFiles, err := findAgentsFiles(opts)
	if err != nil {
		return false, err
	}

	changed := false

	// Sync: ensure each target references existing AGENTS.md
	for _, agentsPath := range agentsFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)

			modified, err := syncFile(targetPath, t.ref, opts.Check)
			if err != nil {
				return changed, fmt.Errorf("sync %s: %w", targetPath, err)
			}
			if modified {
				changed = true
			}
		}
	}

	// Cleanup: remove references for deleted AGENTS.md
	for _, agentsPath := range deletedFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)

			modified, err := removeRef(targetPath, t.ref, opts.Check)
			if err != nil {
				return changed, fmt.Errorf("cleanup %s: %w", targetPath, err)
			}
			if modified {
				changed = true
			}
		}
	}

	// Full cleanup: scan entire repo for stale references (only in --all mode)
	if opts.All {
		cleanupChanged, err := cleanupStaleTargets(agentsFiles, targets, opts.Check)
		if err != nil {
			return changed, err
		}
		changed = changed || cleanupChanged
	}

	return changed, nil
}

// cleanupStaleTargets removes AGENTS.md references from target files where the
// sibling AGENTS.md no longer exists.
func cleanupStaleTargets(agentsFiles []string, targets []target, check bool) (bool, error) {
	// Build a set of directories that have AGENTS.md
	agentsDirs := make(map[string]bool)
	for _, path := range agentsFiles {
		agentsDirs[filepath.Dir(path)] = true
	}

	targetFiles, err := findTargetFiles(targets)
	if err != nil {
		return false, err
	}

	changed := false
	for _, f := range targetFiles {
		dir := filepath.Dir(f.path)
		if agentsDirs[dir] {
			continue // AGENTS.md still exists, skip
		}

		modified, err := removeRef(f.path, f.ref, check)
		if err != nil {
			return changed, fmt.Errorf("cleanup %s: %w", f.path, err)
		}
		if modified {
			changed = true
		}
	}

	return changed, nil
}
