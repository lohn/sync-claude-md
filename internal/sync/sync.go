// Package sync provides the core logic for synchronizing AGENTS.md to the
// per-agent instruction files CLAUDE.md and GEMINI.md.
package sync

import (
	"fmt"
	"path/filepath"
)

// Options controls the behavior of Run.
type Options struct {
	All       bool     // scan entire repository
	Check     bool     // dry-run, only validate
	Files     []string // explicit file list (from pre-commit args)
	Claude    bool     // sync CLAUDE.md
	Gemini    bool     // sync GEMINI.md
	PreCommit bool     // pre-commit subcommand: verify against the git index
	Stage     bool     // pre-commit: git add the synced target files
	Force     bool     // pre-commit: write even if a target has unstaged changes
}

// Run executes the synchronization.
// Returns true if any target file was modified.
func Run(opts Options) (bool, error) {
	actions, err := planActions(opts)
	if err != nil {
		return false, err
	}

	// In check mode we only report whether changes are needed.
	if opts.Check {
		for _, a := range actions {
			if a.modifies() {
				return true, nil
			}
		}
		return false, nil
	}

	return applyActions(actions)
}

// planActions computes every mutation needed to bring the selected targets in
// sync with their sibling AGENTS.md files, without writing anything. Targets
// that are git-ignored in pre-commit mode are skipped entirely (no-op).
func planActions(opts Options) ([]plannedAction, error) {
	targets := resolveTargets(opts)

	agentsFiles, deletedFiles, err := findAgentsFiles(opts)
	if err != nil {
		return nil, err
	}

	var actions []plannedAction

	// Sync: ensure each target references an existing AGENTS.md.
	for _, agentsPath := range agentsFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)
			if opts.PreCommit && isIgnored(targetPath) {
				continue
			}
			a, err := planSync(targetPath, t.ref)
			if err != nil {
				return nil, fmt.Errorf("plan sync %s: %w", targetPath, err)
			}
			actions = append(actions, a)
		}
	}

	// Cleanup: remove references for deleted AGENTS.md.
	for _, agentsPath := range deletedFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)
			if opts.PreCommit && isIgnored(targetPath) {
				continue
			}
			a, err := planCleanup(targetPath, t.ref)
			if err != nil {
				return nil, fmt.Errorf("plan cleanup %s: %w", targetPath, err)
			}
			actions = append(actions, a)
		}
	}

	// Full cleanup: scan entire repo for stale references (only in --all mode).
	if opts.All {
		stale, err := planStaleTargets(agentsFiles, targets)
		if err != nil {
			return nil, err
		}
		actions = append(actions, stale...)
	}

	return actions, nil
}

// applyActions writes every planned mutation to disk in order, returning whether
// anything actually changed. A mid-way error leaves earlier writes in place; the
// planner has already decided the full set, so partial application only happens
// on a genuine I/O failure.
func applyActions(actions []plannedAction) (bool, error) {
	changed := false
	for _, a := range actions {
		if !a.modifies() {
			continue
		}
		if err := applyAction(a); err != nil {
			return changed, fmt.Errorf("apply %s: %w", a.path, err)
		}
		changed = true
	}
	return changed, nil
}

// planStaleTargets returns cleanup actions for target files whose sibling
// AGENTS.md no longer exists.
func planStaleTargets(agentsFiles []string, targets []target) ([]plannedAction, error) {
	// Build a set of directories that have AGENTS.md.
	agentsDirs := make(map[string]bool)
	for _, path := range agentsFiles {
		agentsDirs[filepath.Dir(path)] = true
	}

	targetFiles, err := findTargetFiles(targets)
	if err != nil {
		return nil, err
	}

	var actions []plannedAction
	for _, f := range targetFiles {
		dir := filepath.Dir(f.path)
		if agentsDirs[dir] {
			continue // AGENTS.md still exists, skip
		}

		a, err := planCleanup(f.path, f.ref)
		if err != nil {
			return nil, fmt.Errorf("plan cleanup %s: %w", f.path, err)
		}
		actions = append(actions, a)
	}

	return actions, nil
}
