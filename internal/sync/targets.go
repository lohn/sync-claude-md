package sync

// target describes a per-agent instruction file that mirrors a sibling
// AGENTS.md. Each agent expects a slightly different import line (see the
// claudeTarget/geminiTarget definitions in constants.go).
type target struct {
	filename string // e.g. "CLAUDE.md"
	ref      string // import line referencing the sibling AGENTS.md
}

// targetFile is a discovered instruction file on disk paired with the
// reference line of the target it belongs to.
type targetFile struct {
	path string
	ref  string
}

// resolveTargets returns the target files selected by opts. CLAUDE.md is the
// primary target; GEMINI.md is opt-in. The default decision (CLAUDE.md unless
// --no-claude, GEMINI.md only with --gemini) is made by the CLI, so this maps
// the booleans directly.
func resolveTargets(opts Options) []target {
	var targets []target
	if opts.Claude {
		targets = append(targets, claudeTarget)
	}
	if opts.Gemini {
		targets = append(targets, geminiTarget)
	}
	return targets
}
