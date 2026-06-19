package sync

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// axis identifies which kind of pre-commit guarantee a violation breaks.
type axis int

const (
	// axisDestroy: the tool would overwrite a target file that has unstaged
	// changes, destroying the user's in-progress work. Cleared by --force.
	axisDestroy axis = iota
	// axisSync: the target's reference state is not reflected in the git index,
	// so the sync would not be part of the commit. Cleared by --stage (or a
	// manual git add).
	axisSync
)

// violation records a single pre-commit problem for a target file.
type violation struct {
	path string
	axis axis
}

// PreCommitResult reports the outcome of a pre-commit run so the CLI can choose
// messages and the exit code without re-touching git.
type PreCommitResult struct {
	// DestroyPaths are existing target files with unstaged changes that the sync
	// would overwrite. Populated only when they block the run (no --force); the
	// run wrote nothing in that case.
	DestroyPaths []string
	// SyncPaths are target files whose reference state is not reflected in the
	// git index. Populated only when they still need staging after the run
	// (no --stage).
	SyncPaths []string
	// Wrote and Staged record what the run did.
	Wrote  bool
	Staged bool
}

// RunPreCommit plans the sync, enforces the destroy-protection and index-sync
// guarantees, writes the changes, and optionally stages them. It performs all
// git and filesystem work; the caller maps the result to messages and an exit
// code.
func RunPreCommit(opts Options) (PreCommitResult, error) {
	var res PreCommitResult

	actions, err := planActions(opts)
	if err != nil {
		return res, err
	}

	destroy, sync, err := CheckPreCommit(actions, opts)
	if err != nil {
		return res, err
	}

	// Destroy protection: refuse to clobber unstaged work unless forced.
	if len(destroy) > 0 && !opts.Force {
		res.DestroyPaths = paths(destroy)
		return res, nil
	}

	wrote, err := applyActions(actions)
	if err != nil {
		return res, err
	}
	res.Wrote = wrote

	// Paths that need staging: everything the run wrote, plus any target whose
	// index state is still out of sync (e.g. a file already on disk with the
	// reference but never staged).
	stagePaths := dedup(append(modifyingPaths(actions), paths(sync)...))

	if opts.Stage {
		if err := gitAdd(stagePaths...); err != nil {
			return res, err
		}
		res.Staged = true
		return res, nil
	}

	res.SyncPaths = paths(sync)
	return res, nil
}

// modifyingPaths returns the paths of actions that change the filesystem.
func modifyingPaths(actions []plannedAction) []string {
	var out []string
	for _, a := range actions {
		if a.modifies() {
			out = append(out, a.path)
		}
	}
	return out
}

// paths extracts the path from each violation.
func paths(vs []violation) []string {
	var out []string
	for _, v := range vs {
		out = append(out, v.path)
	}
	return out
}

// dedup returns the input with duplicates removed, preserving first-seen order.
func dedup(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// CheckPreCommit inspects the planned actions and the git index and returns the
// destroy-protection (axisDestroy) and index-sync (axisSync) violations.
//
// axisDestroy is derived from the planned actions: any write to an existing file
// that has unstaged changes would clobber the user's work.
//
// axisSync is derived from the git index: for every selected target whose
// sibling AGENTS.md exists the index must contain the reference, and for every
// target whose AGENTS.md was deleted the index must not contain it.
func CheckPreCommit(actions []plannedAction, opts Options) (destroy, sync []violation, err error) {
	for _, a := range actions {
		if !a.modifies() || a.kind == actionCreate {
			// A create only happens when the file is absent, so there is
			// nothing to destroy.
			continue
		}
		dirty, derr := hasUnstagedChanges(a.path)
		if derr != nil {
			return nil, nil, derr
		}
		if dirty {
			destroy = append(destroy, violation{path: a.path, axis: axisDestroy})
		}
	}

	targets := resolveTargets(opts)
	agentsFiles, deletedFiles, ferr := findAgentsFiles(opts)
	if ferr != nil {
		return nil, nil, ferr
	}

	// AGENTS.md present: the index must carry the reference.
	for _, agentsPath := range agentsFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)
			if isIgnored(targetPath) {
				continue
			}
			has, herr := indexHasRef(targetPath, t.ref)
			if herr != nil {
				return nil, nil, herr
			}
			if !has {
				sync = append(sync, violation{path: targetPath, axis: axisSync})
			}
		}
	}

	// AGENTS.md deleted: the index must no longer carry the reference.
	for _, agentsPath := range deletedFiles {
		dir := filepath.Dir(agentsPath)
		for _, t := range targets {
			targetPath := filepath.Join(dir, t.filename)
			if isIgnored(targetPath) {
				continue
			}
			has, herr := indexHasRef(targetPath, t.ref)
			if herr != nil {
				return nil, nil, herr
			}
			if has {
				sync = append(sync, violation{path: targetPath, axis: axisSync})
			}
		}
	}

	return destroy, sync, nil
}

// isIgnored reports whether path is git-ignored. A tracked file is never
// reported as ignored, matching git's own behavior.
func isIgnored(path string) bool {
	// check-ignore exits 0 when the path is ignored, 1 when it is not.
	return gitProbe("check-ignore", "-q", "--", path)
}

// hasUnstagedChanges reports whether path has changes in the working tree that
// are not staged: a fully- or partially-unstaged modification/deletion, or an
// untracked file. A path that is clean or fully staged returns false.
func hasUnstagedChanges(path string) (bool, error) {
	out, err := gitOutput("status", "--porcelain", "--", path)
	if err != nil {
		return false, err
	}
	status := strings.TrimRight(string(out), "\n")
	if status == "" {
		return false, nil // clean or fully staged
	}
	// Porcelain v1 format: "XY <path>". X is the index column, Y the worktree
	// column. Untracked files are "??". Unstaged changes exist when the
	// worktree column is dirty or the file is untracked.
	if strings.HasPrefix(status, "??") {
		return true, nil
	}
	if len(status) >= 2 && status[1] != ' ' {
		return true, nil
	}
	return false, nil
}

// indexHasRef reports whether the staged (index) content of path contains ref
// on its own line. A path absent from the index returns false.
func indexHasRef(path, ref string) (bool, error) {
	out, err := gitOutput("cat-file", "blob", ":"+path)
	if err != nil {
		// Not in the index (untracked or staged deletion): no reference there.
		return false, nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == ref {
			return true, nil
		}
	}
	return false, nil
}

// gitAdd stages the given paths. Paths that are neither present on disk nor
// tracked in the index are skipped: `git add` errors on such a pathspec, which
// would otherwise break --stage when cleanup deleted an untracked target.
func gitAdd(paths ...string) error {
	var stageable []string
	for _, p := range paths {
		if isStageable(p) {
			stageable = append(stageable, p)
		}
	}
	if len(stageable) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, stageable...)
	return execGit(args...).Run()
}

// isStageable reports whether `git add path` would have something to stage:
// the path exists on disk (add/modify) or is tracked in the index (a deletion
// to stage). An untracked path that was also removed from disk has nothing to
// stage and would make `git add` fail with a pathspec error.
func isStageable(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	// Absent on disk: only stageable if git is tracking it (staged deletion).
	return gitProbe("ls-files", "--error-unmatch", "--", path)
}

// gitOutput runs a git command and returns its stdout, suppressing stderr so
// expected failures (e.g. cat-file on a missing path) stay quiet.
func gitOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.Bytes(), err
}

// gitProbe runs a git command for its exit status only, returning true on a
// zero exit code. stdout and stderr are discarded.
func gitProbe(args ...string) bool {
	return exec.Command("git", args...).Run() == nil
}
