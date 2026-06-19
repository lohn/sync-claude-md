package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lohn/sync-claude-md/internal/sync"
)

// These variables are set by ldflags during build (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const usageHeader = `sync-claude-md keeps each CLAUDE.md in sync with its sibling AGENTS.md.

For every directory that contains an AGENTS.md, it ensures a CLAUDE.md sits
next to it whose first line is "@AGENTS.md". When an AGENTS.md is removed, the
matching reference is dropped, and the CLAUDE.md is deleted if nothing else
remains in it.

CLAUDE.md is synced by default. Pass --gemini to also sync a GEMINI.md
("@./AGENTS.md") in each directory, or --no-claude to skip CLAUDE.md (e.g.
together with --gemini to sync GEMINI.md only).

Usage:
  sync-claude-md [flags] [files...]
  sync-claude-md pre-commit [flags]

With no flags or file arguments, only staged AGENTS.md files are processed,
which is the intended pre-commit hook mode. Any [files...] given take priority
over --all and over staged-file detection.

The "pre-commit" subcommand verifies the result against the git index: it fails
the commit when a target's @AGENTS.md reference is not staged, so the sync is
guaranteed to land in the commit. See "pre-commit --help".

Flags:
`

const usageExamples = `
Examples:
  sync-claude-md                  Sync CLAUDE.md for staged AGENTS.md (pre-commit)
  sync-claude-md --all            Scan the whole repository
  sync-claude-md --check --all    Report drift without writing; exit 1 if any (CI)
  sync-claude-md --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md docs/AGENTS.md   Sync only the given AGENTS.md files
  sync-claude-md pre-commit       Sync staged AGENTS.md and verify against the index
`

const preCommitUsageHeader = `sync-claude-md pre-commit syncs the per-agent files for staged AGENTS.md and
verifies the result against the git index.

It enforces two guarantees:
  - Sync: the @AGENTS.md reference must be staged, so the sync lands in the
    commit. Otherwise the commit is stopped (exit 1) asking you to "git add".
    Pass --stage to stage the synced files automatically instead.
  - Destroy protection: it refuses to overwrite a target file that has unstaged
    changes (which would discard your work). Pass --force to override.

Usage:
  sync-claude-md pre-commit [flags]

Flags:
`

const preCommitUsageExamples = `
Examples:
  sync-claude-md pre-commit          Sync staged AGENTS.md and verify the index
  sync-claude-md pre-commit --stage  Sync and stage the result (exit 0 in one pass)
  sync-claude-md pre-commit --gemini Also verify GEMINI.md alongside CLAUDE.md
`

func usage() {
	fmt.Fprint(os.Stderr, usageHeader)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(os.Stderr, "  --%-11s %s\n", f.Name, f.Usage)
	})
	fmt.Fprint(os.Stderr, usageExamples)
}

// preCommitUsage prints the pre-commit subcommand help in the same style as the
// top-level usage: a header, one aligned line per flag (collapsing shorthand
// aliases onto their long form), then examples.
func preCommitUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, preCommitUsageHeader)
	// Skip shorthand aliases so each flag is listed once; note the alias inline.
	aliases := map[string]string{"stage": "-S", "force": "-f"}
	fs.VisitAll(func(f *flag.Flag) {
		if len(f.Name) == 1 {
			return // shorthand alias, shown alongside its long form
		}
		name := "--" + f.Name
		if alias, ok := aliases[f.Name]; ok {
			name += ", " + alias
		}
		fmt.Fprintf(os.Stderr, "  %-16s %s\n", name, f.Usage)
	})
	fmt.Fprint(os.Stderr, preCommitUsageExamples)
}

func main() {
	// Subcommand dispatch: "pre-commit" has its own flag set.
	if len(os.Args) > 1 && os.Args[1] == "pre-commit" {
		os.Exit(runPreCommit(os.Args[2:]))
	}

	flag.Usage = usage

	var (
		all         = flag.Bool("all", false, "Scan the entire repository instead of only staged files")
		check       = flag.Bool("check", false, "Report whether changes are needed without writing them; exit 1 on drift")
		geminiFlag  = flag.Bool("gemini", false, "Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md")
		noClaude    = flag.Bool("no-claude", false, "Do not sync CLAUDE.md (use with --gemini to sync GEMINI.md only)")
		versionFlag = flag.Bool("version", false, "Print version information and exit")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("sync-claude-md %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	claude, gemini, ok := selectTargets(*noClaude, *geminiFlag)
	if !ok {
		os.Exit(1)
	}

	opts := sync.Options{
		All:    *all,
		Check:  *check,
		Files:  flag.Args(),
		Claude: claude,
		Gemini: gemini,
	}

	changed, err := sync.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if changed {
		if *check {
			fmt.Fprintln(os.Stderr, "agent instruction files are out of sync")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "agent instruction files updated. Please re-stage changes.")
		os.Exit(1)
	}
}

// runPreCommit handles the "pre-commit" subcommand and returns the process exit
// code.
func runPreCommit(args []string) int {
	fs := flag.NewFlagSet("pre-commit", flag.ExitOnError)
	fs.Usage = func() { preCommitUsage(fs) }
	var (
		geminiFlag = fs.Bool("gemini", false, "Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md")
		noClaude   = fs.Bool("no-claude", false, "Do not sync CLAUDE.md (use with --gemini to sync GEMINI.md only)")
		stage      = fs.Bool("stage", false, "git add the synced target files (exit 0 in one pass)")
		force      = fs.Bool("force", false, "Overwrite target files even if they have unstaged changes")
	)
	fs.BoolVar(stage, "S", false, "Shorthand for --stage")
	fs.BoolVar(force, "f", false, "Shorthand for --force")
	_ = fs.Parse(args)

	claude, gemini, ok := selectTargets(*noClaude, *geminiFlag)
	if !ok {
		return 1
	}

	opts := sync.Options{
		Files:     fs.Args(),
		Claude:    claude,
		Gemini:    gemini,
		PreCommit: true,
		Stage:     *stage,
		Force:     *force,
	}

	res, err := sync.RunPreCommit(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Destroy protection blocked the run: nothing was written.
	if len(res.DestroyPaths) > 0 {
		fmt.Fprintln(os.Stderr, "error: refusing to overwrite files with unstaged changes:")
		for _, p := range res.DestroyPaths {
			fmt.Fprintf(os.Stderr, "  %s\n", p)
		}
		fmt.Fprintln(os.Stderr, "stage or discard your changes, or pass --force to overwrite.")
		return 1
	}

	// References not reflected in the index: the sync would miss the commit.
	if len(res.SyncPaths) > 0 {
		fmt.Fprintln(os.Stderr, "agent instruction files updated but not staged. Run:")
		for _, p := range res.SyncPaths {
			fmt.Fprintf(os.Stderr, "  git add %s\n", p)
		}
		fmt.Fprintln(os.Stderr, "then commit again (or pass --stage to stage automatically).")
		return 1
	}

	return 0
}

// selectTargets resolves the CLAUDE.md/GEMINI.md selection from the flags.
// CLAUDE.md is on unless --no-claude; GEMINI.md is opt-in via --gemini.
// --no-claude without --gemini leaves nothing to do and reports an error.
func selectTargets(noClaude, gemini bool) (claude, geminiOut, ok bool) {
	claude = !noClaude
	if !claude && !gemini {
		fmt.Fprintln(os.Stderr, "error: nothing to sync (--no-claude without --gemini)")
		return false, false, false
	}
	return claude, gemini, true
}
