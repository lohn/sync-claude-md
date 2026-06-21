package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/lohn/sync-claude-md/internal/sync"
)

// These variables are set by ldflags during release builds (see
// .goreleaser.yaml). Other build paths — notably `go install
// .../cmd/sync-claude-md@latest`, which never runs goreleaser — leave them at
// these defaults, so init() falls back to the module version and VCS
// revision Go embeds in the binary automatically. date is never derived this
// way (see versionFromBuildInfo) and always keeps its default outside a
// goreleaser build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if version != "dev" {
		return // already set via -ldflags
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		version, commit = versionFromBuildInfo(info, version, commit)
	}
}

// versionFromBuildInfo derives version/commit from the build info Go embeds
// in every binary, falling back to the given defaults for whichever field it
// has no data for. info.Main.Version is the resolved module version (e.g.
// "v1.0.0") when installed via `go install pkg@version`, but "(devel)" when
// built from a local checkout without VCS stamping; the vcs.revision
// setting, conversely, is only present when built from an actual VCS
// checkout (a plain `go build` in a git clone), not when installed from the
// module proxy — the two sources are independent and either may be missing.
// date is intentionally not derived here: build info has no actual
// build-time field, only vcs.time (the timestamp of the vcs.revision
// commit), and using that would mislabel a commit time as the binary's
// "built:" time in --version output.
func versionFromBuildInfo(info *debug.BuildInfo, version, commit string) (string, string) {
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			commit = s.Value
		}
	}
	return version, commit
}

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

Run "sync-claude-md <command> -h" for the flags of a specific command.

Other flags:
  --version    Print version information and exit

Examples:
  sync-claude-md sync                  Sync staged AGENTS.md and verify against the git index
  sync-claude-md sync --stage          Sync and stage the result (exit 0 in one pass)
  sync-claude-md sync --all            Scan the whole repository
  sync-claude-md check --all           Report drift without writing; exit 1 if any (CI)
  sync-claude-md sync --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md sync docs/AGENTS.md   Sync only the given AGENTS.md files
`

const syncUsageHeader = `sync-claude-md sync creates, updates, or cleans up CLAUDE.md (and, with
--gemini, GEMINI.md) so each one references its sibling AGENTS.md.

With no file arguments, only staged AGENTS.md files are processed, which is
the intended git-hook use. Pass --all to scan the whole repository instead.
Outside a git repository "staged" is meaningless, so the default falls back
to a full scan too. Any [files...] given take priority over both.

It enforces three guarantees:
  - Destroy protection: it refuses to overwrite an existing target file that
    has unstaged changes, which would discard your work, and exits 1 without
    writing. Pass --force to overwrite anyway.
  - Outside a git repository, it refuses to write anything at all — even a
    brand-new file — since there is no git history to recover from, and
    exits 1. Pass --force to write anyway.
  - Index sync (inside a git repository only): the @AGENTS.md reference must
    be staged, so the sync actually lands in the next commit. If it is not
    (including a freshly created but untracked CLAUDE.md), it exits 1 and
    asks you to "git add" the file. Pass --stage to stage the synced files
    automatically and succeed in a single pass.

Usage:
  sync-claude-md sync [flags] [files...]

Flags:
`

const syncUsageExamples = `
Examples:
  sync-claude-md sync                  Sync staged AGENTS.md and verify against the git index
  sync-claude-md sync --stage          Sync and stage the result (exit 0 in one pass)
  sync-claude-md sync --all            Scan the whole repository
  sync-claude-md sync --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md sync --force          Overwrite targets even with unstaged changes
  sync-claude-md sync --no-ignore      Also process git-ignored target files
  sync-claude-md sync --fail-on-change Exit 1 if anything was written, even when staged
  sync-claude-md sync docs/AGENTS.md   Sync only the given AGENTS.md files
`

const checkUsageHeader = `sync-claude-md check reports whether CLAUDE.md (and, with --gemini,
GEMINI.md) is in sync with its sibling AGENTS.md — on disk and, inside a git
repository, in the git index too — without writing anything.

Usage:
  sync-claude-md check [flags] [files...]

Flags:
`

const checkUsageExamples = `
Examples:
  sync-claude-md check --all     Report drift without writing; exit 1 if any (CI)
  sync-claude-md check --gemini  Also check GEMINI.md
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
	noIgnore bool
}

func bindCommonFlags(fs *flag.FlagSet, c *commonFlags) {
	fs.BoolVar(&c.all, "all", false, "Scan the entire repository instead of only staged files")
	fs.BoolVar(&c.gemini, "gemini", false, "Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md")
	fs.BoolVar(&c.noClaude, "no-claude", false, "Do not sync CLAUDE.md (use with --gemini to sync GEMINI.md only)")
	fs.BoolVar(&c.noIgnore, "no-ignore", false, "Also process target files that are git-ignored")
}

// runSync handles the "sync" subcommand and returns the process exit code.
func runSync(args []string) int {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	var c commonFlags
	bindCommonFlags(fs, &c)
	var (
		force        bool
		stage        bool
		failOnChange bool
	)
	fs.BoolVar(&force, "force", false, "Overwrite targets with unstaged changes, or write at all outside a git repository")
	fs.BoolVar(&force, "f", false, "Shorthand for --force")
	fs.BoolVar(&stage, "stage", false, "git add the synced target files (inside a git repository only)")
	fs.BoolVar(&stage, "S", false, "Shorthand for --stage")
	fs.BoolVar(&failOnChange, "fail-on-change", false, "Exit 1 if any file was written, even after a successful sync/stage")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, syncUsageHeader)
		printFlags(fs, map[string]string{"force": "-f", "stage": "-S"})
		fmt.Fprint(os.Stderr, syncUsageExamples)
	}
	_ = fs.Parse(args)

	claude, gemini, ok := selectTargets(c.noClaude, c.gemini)
	if !ok {
		return 1
	}

	opts := sync.Options{
		All:      c.all,
		Files:    fs.Args(),
		Claude:   claude,
		Gemini:   gemini,
		Force:    force,
		Stage:    stage,
		NoIgnore: c.noIgnore,
	}

	res, err := sync.Run(opts)
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

	// Outside a git repository: nothing was written without explicit confirmation.
	if len(res.NoGitPaths) > 0 {
		fmt.Fprintln(os.Stderr, "error: refusing to write outside a git repository:")
		for _, p := range res.NoGitPaths {
			fmt.Fprintf(os.Stderr, "  %s\n", p)
		}
		fmt.Fprintln(os.Stderr, "pass --force to write anyway (no git history to recover from outside a repository).")
		return 1
	}

	// References not reflected in the index: the sync would miss the next commit.
	// This can happen with no write this run too (e.g. the target was already
	// correct on disk from an earlier run but never staged), so the message
	// talks about staging state rather than claiming a write just happened.
	if len(res.SyncPaths) > 0 {
		fmt.Fprintln(os.Stderr, "agent instruction files are not staged. Run:")
		for _, p := range res.SyncPaths {
			fmt.Fprintf(os.Stderr, "  git add -- %s\n", p)
		}
		fmt.Fprintln(os.Stderr, "then try again (or pass --stage to stage automatically).")
		return 1
	}

	if failOnChange && res.Wrote {
		fmt.Fprintln(os.Stderr, "agent instruction files were updated.")
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
		All:      c.all,
		Check:    true,
		Files:    fs.Args(),
		Claude:   claude,
		Gemini:   gemini,
		NoIgnore: c.noIgnore,
	}

	res, err := sync.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if res.Changed {
		fmt.Fprintln(os.Stderr, "agent instruction files are out of sync")
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
