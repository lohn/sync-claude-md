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

With no flags or file arguments, only staged AGENTS.md files are processed,
which is the intended pre-commit hook mode. Any [files...] given take priority
over --all and over staged-file detection.

Flags:
`

const usageExamples = `
Examples:
  sync-claude-md                  Sync CLAUDE.md for staged AGENTS.md (pre-commit)
  sync-claude-md --all            Scan the whole repository
  sync-claude-md --check --all    Report drift without writing; exit 1 if any (CI)
  sync-claude-md --gemini         Also sync GEMINI.md alongside CLAUDE.md
  sync-claude-md docs/AGENTS.md   Sync only the given AGENTS.md files
`

func usage() {
	fmt.Fprint(os.Stderr, usageHeader)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(os.Stderr, "  --%-11s %s\n", f.Name, f.Usage)
	})
	fmt.Fprint(os.Stderr, usageExamples)
}

func main() {
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

	// CLAUDE.md is synced by default; --no-claude opts out. GEMINI.md is opt-in
	// via --gemini. --no-claude alone leaves nothing to do.
	claude := !*noClaude
	gemini := *geminiFlag
	if !claude && !gemini {
		fmt.Fprintln(os.Stderr, "error: nothing to sync (--no-claude without --gemini)")
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
