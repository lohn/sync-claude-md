package sync

import (
	"os"
	"strings"
)

// syncFile ensures the target file exists with the given AGENTS.md reference.
func syncFile(targetPath, ref string, check bool) (bool, error) {
	exists := false
	if _, err := os.Stat(targetPath); err == nil {
		exists = true
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if !exists {
		if check {
			return true, nil
		}
		return true, createTarget(targetPath, ref)
	}

	return updateTarget(targetPath, ref, check)
}

// createTarget creates a new target file containing only the AGENTS.md reference.
func createTarget(targetPath, ref string) error {
	return os.WriteFile(targetPath, []byte(ref+"\n"), targetFileMode)
}

// updateTarget ensures the target file references AGENTS.md. The reference may
// live anywhere in the file; only its absence triggers a write, in which case
// it is inserted at the top.
func updateTarget(targetPath, ref string, check bool) (bool, error) {
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")

	// Already references AGENTS.md somewhere? Then nothing to do.
	for _, line := range lines {
		if strings.TrimSpace(line) == ref {
			return false, nil
		}
	}

	if check {
		return true, nil
	}

	// Insert the reference at the top, dropping any leading blank lines so we do
	// not accumulate empty lines, and separating it from existing content with
	// a single blank line.
	firstNonEmpty := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = i
			break
		}
	}
	rest := lines[firstNonEmpty:]

	newLines := []string{ref}
	if len(rest) > 0 {
		newLines = append(newLines, "")
	}
	newLines = append(newLines, rest...)

	newContent := strings.Join(newLines, "\n")
	return true, os.WriteFile(targetPath, []byte(newContent), targetFileMode)
}

// removeRef removes the AGENTS.md reference from a target file. A line is a
// reference when, trimmed of surrounding whitespace, it equals ref exactly; such
// lines are removed wherever they appear (not only at the top), mirroring
// updateTarget, which treats the reference as present anywhere. A single blank
// line immediately following each removed reference is dropped too, so we do not
// accumulate empty lines. If the file becomes empty after removal, it is deleted.
func removeRef(targetPath, ref string, check bool) (bool, error) {
	content, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		// No target file means there is no reference to remove. This happens
		// when an AGENTS.md is deleted in a directory that never had this
		// target (e.g. GEMINI.md when only CLAUDE.md was synced).
		return false, nil
	}
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")

	// The reference must appear as a standalone line somewhere; otherwise there
	// is nothing to remove.
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == ref {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	if check {
		return true, nil
	}

	// Drop every standalone reference line, plus one blank line immediately
	// following each, to avoid leaving accumulated empty lines behind.
	var newLines []string
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == ref {
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++
			}
			continue
		}
		newLines = append(newLines, lines[i])
	}

	// Check if file is now empty (only whitespace left)
	hasContent := false
	for _, line := range newLines {
		if strings.TrimSpace(line) != "" {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return true, os.Remove(targetPath)
	}

	newContent := strings.Join(newLines, "\n")
	return true, os.WriteFile(targetPath, []byte(newContent), targetFileMode)
}
