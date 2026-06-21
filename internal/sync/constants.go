package sync

import "os"

// targetFileMode is the permission used when writing target files.
const targetFileMode os.FileMode = 0o644

// maxTargetFileSize is the largest target file (CLAUDE.md/GEMINI.md) this
// tool will read. A legitimate target file holds only a short reference line
// plus whatever the user prepends, so anything past this is almost certainly
// not one — refuse outright rather than risk loading an arbitrarily large
// file into memory.
const maxTargetFileSize = 10 * 1024 * 1024 // 10 MiB

// maxLineLength is the longest line considered when checking whether a
// target-file line is the bare reference (e.g. "@AGENTS.md", 10 bytes, or
// "@./AGENTS.md", 12 bytes). A line longer than this cannot equal the
// reference after trimming, so it is skipped without paying for a full
// TrimSpace/compare — generous enough for any plausible amount of incidental
// whitespace around the reference, while staying tiny relative to
// maxTargetFileSize so one pathological line can't be expensive to scan.
const maxLineLength = 1024 // 1 KiB

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
