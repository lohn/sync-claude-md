// Package sync provides the core logic for synchronizing AGENTS.md to the
// per-agent instruction files CLAUDE.md and GEMINI.md.
package sync

import (
	"fmt"
	"path/filepath"
)

// Options controls the behavior of Run.
type Options struct {
	All      bool     // scan entire repository
	Check    bool     // dry-run, only report drift; never writes or stages
	Files    []string // explicit file list
	Claude   bool     // sync CLAUDE.md
	Gemini   bool     // sync GEMINI.md
	Stage    bool     // git add the synced/cleaned-up target files (inside a git repository only)
	Force    bool     // write/overwrite even if a target has unstaged changes, or outside a git repository
	NoIgnore bool     // also process target files that are git-ignored (default: skip them)
}

// Result reports what Run did, or — in Check mode — what it would do.
type Result struct {
	// Changed is set by Check mode: true if any selected target's on-disk
	// content, or (inside a git repository) git-index state, differs from what
	// its sibling AGENTS.md implies. Always false outside Check mode.
	Changed bool

	// DestroyPaths are existing target files with unstaged changes that the run
	// refused to overwrite. Populated only when they block the run (no
	// Options.Force); the run wrote nothing in that case. Always empty in Check
	// mode and outside a git repository (see NoGitPaths instead).
	DestroyPaths []string

	// NoGitPaths are target files that would have been written outside a git
	// repository. Populated only when they block the run (no Options.Force);
	// the run wrote nothing in that case. Always empty in Check mode and inside
	// a git repository (see DestroyPaths instead).
	NoGitPaths []string

	// SyncPaths are target files whose git-index state does not match the
	// desired reference state. Populated only when they still need staging
	// after the run (no Options.Stage). Always empty in Check mode and outside
	// a git repository.
	SyncPaths []string

	// Wrote and Staged record what the run did. Always false in Check mode.
	Wrote  bool
	Staged bool
}

// Run executes the synchronization, or, in Check mode, only inspects it.
//
// Before writing, it refuses to overwrite an existing target file that has
// unstaged changes, which would discard work that is not yet staged
// (Result.DestroyPaths); pass Options.Force to skip this check. Outside a git
// repository "unstaged" cannot be evaluated at all, so it instead refuses to
// write anything — including a brand-new file — since there is no git
// history to recover from (Result.NoGitPaths); pass Options.Force to write
// anyway. Inside a git repository, it additionally verifies the result
// against the git index — the target's reference state must be staged for a
// commit made now to actually include it (Result.SyncPaths); pass
// Options.Stage to stage the written files automatically. All of this is
// skipped in Check mode, which never writes.
func Run(opts Options) (Result, error) {
	actions, err := planActions(opts)
	if err != nil {
		return Result{}, err
	}

	git := inGitRepo()

	if opts.Check {
		changed := anyModifies(actions)
		if git {
			violations, err := checkIndexSync(actions)
			if err != nil {
				return Result{}, err
			}
			if len(violations) > 0 {
				changed = true
			}
		}
		return Result{Changed: changed}, nil
	}

	if !opts.Force {
		if !git {
			if anyModifies(actions) {
				return Result{NoGitPaths: modifyingPaths(actions)}, nil
			}
		} else {
			destroy, err := checkDestroy(actions)
			if err != nil {
				return Result{}, err
			}
			if len(destroy) > 0 {
				return Result{DestroyPaths: paths(destroy)}, nil
			}
		}
	}

	wrote, err := applyActions(actions)
	if err != nil {
		return Result{}, err
	}

	var syncViolations []violation
	if git {
		syncViolations, err = checkIndexSync(actions)
		if err != nil {
			return Result{}, err
		}
	}

	if opts.Stage && git {
		stagePaths := dedup(append(modifyingPaths(actions), paths(syncViolations)...))
		if err := gitAdd(stagePaths...); err != nil {
			return Result{}, err
		}
		return Result{Wrote: wrote, Staged: true}, nil
	}

	return Result{Wrote: wrote, SyncPaths: paths(syncViolations)}, nil
}

// planActions computes every mutation needed to bring the selected targets in
// sync with their sibling AGENTS.md files, without writing anything. Targets
// that are git-ignored are skipped entirely (no-op) unless Options.NoIgnore.
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
			if !opts.NoIgnore && isIgnored(targetPath) {
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
			if !opts.NoIgnore && isIgnored(targetPath) {
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
		stale, err := planStaleTargets(agentsFiles, targets, opts.NoIgnore)
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
// AGENTS.md no longer exists. Git-ignored targets are skipped unless noIgnore,
// matching planActions's main sync/cleanup loops.
func planStaleTargets(agentsFiles []string, targets []target, noIgnore bool) ([]plannedAction, error) {
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
		if !noIgnore && isIgnored(f.path) {
			continue
		}

		a, err := planCleanup(f.path, f.ref)
		if err != nil {
			return nil, fmt.Errorf("plan cleanup %s: %w", f.path, err)
		}
		actions = append(actions, a)
	}

	return actions, nil
}
