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
  sync-claude-md docs/AGENTS.md   Sync only the given AGENTS.md files
`

func usage() {
	fmt.Fprint(os.Stderr, usageHeader)
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(os.Stderr, "  --%-9s %s\n", f.Name, f.Usage)
	})
	fmt.Fprint(os.Stderr, usageExamples)
}

func main() {
	flag.Usage = usage

	var (
		all         = flag.Bool("all", false, "Scan the entire repository instead of only staged files")
		check       = flag.Bool("check", false, "Report whether changes are needed without writing them; exit 1 on drift")
		versionFlag = flag.Bool("version", false, "Print version information and exit")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("sync-claude-md %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	opts := sync.Options{
		All:   *all,
		Check: *check,
		Files: flag.Args(),
	}

	changed, err := sync.Run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if changed {
		if *check {
			fmt.Fprintln(os.Stderr, "CLAUDE.md files are out of sync")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "CLAUDE.md files updated. Please re-stage changes.")
		os.Exit(1)
	}
}
