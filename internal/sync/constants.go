package sync

import "os"

// targetFileMode is the permission used when writing target files.
const targetFileMode os.FileMode = 0o644

// Target instruction files and the import line each agent expects: Claude Code
// reads "@AGENTS.md" while Gemini reads "@./AGENTS.md".
var (
	claudeTarget = target{filename: "CLAUDE.md", ref: "@AGENTS.md"}
	geminiTarget = target{filename: "GEMINI.md", ref: "@./AGENTS.md"}
)

// skipDirs are directory names excluded from repository walks.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}
