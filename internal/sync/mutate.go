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

// removeRef removes the AGENTS.md reference line from a target file.
// Only removes the first occurrence at the top of the file (not inline references).
// Also removes immediately following blank lines to prevent empty line accumulation.
// If the file becomes empty after removal, it deletes the file.
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

	// Find the first non-empty line
	firstNonEmpty := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = i
			break
		}
	}

	// If first non-empty line is not the reference, skip
	if firstNonEmpty < 0 || strings.TrimSpace(lines[firstNonEmpty]) != ref {
		return false, nil
	}

	if check {
		return true, nil
	}

	// Remove the reference line and any immediately following blank lines
	var newLines []string
	skipping := true // Start skipping after we pass the reference line
	for i, line := range lines {
		if i < firstNonEmpty {
			newLines = append(newLines, line)
			continue
		}
		if i == firstNonEmpty {
			// Skip the reference line itself
			continue
		}
		if skipping && strings.TrimSpace(line) == "" {
			// Skip blank lines immediately following the reference
			continue
		}
		skipping = false
		newLines = append(newLines, line)
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
