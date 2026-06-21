package main

import (
	"errors"
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

const helpText = `sync-claude-md keeps each CLAUDE.md in sync with its sibling AGENTS.md.

For every directory that contains an AGENTS.md, it ensures a CLAUDE.md sits
next to it whose first line is "@AGENTS.md". When an AGENTS.md is removed, the
matching reference is dropped, and the CLAUDE.md is deleted if nothing else
remains in it.

CLAUDE.md is synced by default. Pass --gemini to also sync a GEMINI.md
("@./AGENTS.md") in each directory, or --no-claude to skip CLAUDE.md (e.g.
together with --gemini to sync GEMINI.md only).

Usage:
  sync-claude-md <command> [flags] [files...]

Commands:
  sync         Create, update, or clean up CLAUDE.md (and/or GEMINI.md)
  check        Report drift without writing; exit 1 if any agent file is out of sync
  pre-commit   Sync staged AGENTS.md and verify the result against the git index

Run "sync-claude-md <command> -h" for the flags of a specific command.

Other flags:
  --version    Print version information and exit

Examples:
  sync-claude-md sync                  Sync CLAUDE.md for staged AGENTS.md
  sync-claude-md sync --all            Scan the whole repository
  sync-claude-md check --all           Report drift without writing; exit 1 if any (CI)
  sync-claude-md sync --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md sync docs/AGENTS.md   Sync only the given AGENTS.md files
  sync-claude-md pre-commit            Sync staged AGENTS.md and verify against the index
`

const syncUsageHeader = `sync-claude-md sync creates, updates, or cleans up CLAUDE.md (and, with
--gemini, GEMINI.md) so each one references its sibling AGENTS.md.

With no file arguments, only staged AGENTS.md files are processed, which is
the intended pre-commit hook mode. Any [files...] given take priority over
--all and over staged-file detection.

It refuses to overwrite an existing target file that has unstaged changes,
which would discard your work. Pass --force to overwrite anyway.

Usage:
  sync-claude-md sync [flags] [files...]

Flags:
`

const syncUsageExamples = `
Examples:
  sync-claude-md sync                  Sync CLAUDE.md for staged AGENTS.md
  sync-claude-md sync --all            Scan the whole repository
  sync-claude-md sync --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md sync --force          Overwrite targets even with unstaged changes
  sync-claude-md sync docs/AGENTS.md   Sync only the given AGENTS.md files
`

const checkUsageHeader = `sync-claude-md check reports whether CLAUDE.md (and, with --gemini,
GEMINI.md) is in sync with its sibling AGENTS.md, without writing anything.

Usage:
  sync-claude-md check [flags] [files...]

Flags:
`

const checkUsageExamples = `
Examples:
  sync-claude-md check --all     Report drift without writing; exit 1 if any (CI)
  sync-claude-md check --gemini  Also check GEMINI.md
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

// printFlags writes one aligned line per flag, collapsing single-letter
// shorthand aliases onto their long form via the aliases map (long name ->
// shorthand, e.g. {"force": "-f"}).
func printFlags(fs *flag.FlagSet, aliases map[string]string) {
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
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		return
	}

	switch os.Args[1] {
	case "-h", "--help", "help":
		fmt.Print(helpText)
	case "--version":
		fmt.Printf("sync-claude-md %s (commit: %s, built: %s)\n", version, commit, date)
	case "sync":
		os.Exit(runSync(os.Args[2:]))
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "pre-commit":
		os.Exit(runPreCommit(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(1)
	}
}

// commonFlags are shared by the sync and check subcommands.
type commonFlags struct {
	all      bool
	gemini   bool
	noClaude bool
}

func bindCommonFlags(fs *flag.FlagSet, c *commonFlags) {
	fs.BoolVar(&c.all, "all", false, "Scan the entire repository instead of only staged files")
	fs.BoolVar(&c.gemini, "gemini", false, "Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md")
	fs.BoolVar(&c.noClaude, "no-claude", false, "Do not sync CLAUDE.md (use with --gemini to sync GEMINI.md only)")
}

// runSync handles the "sync" subcommand and returns the process exit code.
func runSync(args []string) int {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	var c commonFlags
	bindCommonFlags(fs, &c)
	var force bool
	fs.BoolVar(&force, "force", false, "Overwrite target files even if they have unstaged changes")
	fs.BoolVar(&force, "f", false, "Shorthand for --force")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, syncUsageHeader)
		printFlags(fs, map[string]string{"force": "-f"})
		fmt.Fprint(os.Stderr, syncUsageExamples)
	}
	_ = fs.Parse(args)

	claude, gemini, ok := selectTargets(c.noClaude, c.gemini)
	if !ok {
		return 1
	}

	opts := sync.Options{
		All:    c.all,
		Files:  fs.Args(),
		Claude: claude,
		Gemini: gemini,
		Force:  force,
	}

	changed, err := sync.Run(opts)
	if err != nil {
		var destroyErr *sync.DestroyError
		if errors.As(err, &destroyErr) {
			fmt.Fprintln(os.Stderr, "error: refusing to overwrite files with unstaged changes:")
			for _, p := range destroyErr.Paths {
				fmt.Fprintf(os.Stderr, "  %s\n", p)
			}
			fmt.Fprintln(os.Stderr, "stage or discard your changes, or pass --force to overwrite.")
			return 1
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if changed {
		fmt.Fprintln(os.Stderr, "agent instruction files updated. Please re-stage changes.")
		return 1
	}
	return 0
}

// runCheck handles the "check" subcommand and returns the process exit code.
func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	var c commonFlags
	bindCommonFlags(fs, &c)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, checkUsageHeader)
		printFlags(fs, nil)
		fmt.Fprint(os.Stderr, checkUsageExamples)
	}
	_ = fs.Parse(args)

	claude, gemini, ok := selectTargets(c.noClaude, c.gemini)
	if !ok {
		return 1
	}

	opts := sync.Options{
		All:    c.all,
		Check:  true,
		Files:  fs.Args(),
		Claude: claude,
		Gemini: gemini,
	}

	changed, err := sync.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if changed {
		fmt.Fprintln(os.Stderr, "agent instruction files are out of sync")
		return 1
	}
	return 0
}

// runPreCommit handles the "pre-commit" subcommand and returns the process exit
// code.
func runPreCommit(args []string) int {
	fs := flag.NewFlagSet("pre-commit", flag.ExitOnError)
	var (
		geminiFlag = fs.Bool("gemini", false, "Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md")
		noClaude   = fs.Bool("no-claude", false, "Do not sync CLAUDE.md (use with --gemini to sync GEMINI.md only)")
		stage      = fs.Bool("stage", false, "git add the synced target files (exit 0 in one pass)")
		force      = fs.Bool("force", false, "Overwrite target files even if they have unstaged changes")
	)
	fs.BoolVar(stage, "S", false, "Shorthand for --stage")
	fs.BoolVar(force, "f", false, "Shorthand for --force")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, preCommitUsageHeader)
		printFlags(fs, map[string]string{"stage": "-S", "force": "-f"})
		fmt.Fprint(os.Stderr, preCommitUsageExamples)
	}
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
