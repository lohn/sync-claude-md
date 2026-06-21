package sync

import (
	"fmt"
	"os"
	"strings"
)

// actionKind enumerates the mutations the planner can decide on for a single
// target file.
type actionKind int

const (
	actionNone      actionKind = iota // nothing to do
	actionCreate                      // create a new file containing only the reference
	actionUpdate                      // prepend the reference to an existing file
	actionRemoveRef                   // strip the reference(s), keep the rest
	actionDelete                      // remove the file (empty after reference removal)
)

// plannedAction is a single, fully-decided mutation for a target file. The
// planner computes these without touching disk; applyAction performs the write.
// For create/update/removeRef, content holds the exact bytes to write.
type plannedAction struct {
	path    string
	ref     string
	kind    actionKind
	content []byte
	// wantRef is the reference's desired presence once committed. kind alone
	// cannot tell a satisfied sync from a satisfied cleanup (both are
	// actionNone), but the git-index check needs to know which is which.
	wantRef bool
}

// modifies reports whether the action changes anything on disk.
func (a plannedAction) modifies() bool { return a.kind != actionNone }

// planSync decides what (if anything) must change so that targetPath references
// ref. It performs no writes.
func planSync(targetPath, ref string) (plannedAction, error) {
	content, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return plannedAction{path: targetPath, ref: ref, kind: actionCreate, content: []byte(ref + "\n"), wantRef: true}, nil
	}
	if err != nil {
		return plannedAction{}, err
	}

	newContent, changed := withRefPrepended(string(content), ref)
	if !changed {
		return plannedAction{path: targetPath, ref: ref, kind: actionNone, wantRef: true}, nil
	}
	return plannedAction{path: targetPath, ref: ref, kind: actionUpdate, content: []byte(newContent), wantRef: true}, nil
}

// planCleanup decides what (if anything) must change to drop ref from
// targetPath. It performs no writes. A missing file is a no-op so cleanup of a
// deleted AGENTS.md never fails for a target that was never synced. wantRef is
// left at its zero value (false) on every returned action: cleanup always
// wants the reference gone, whether or not anything actually needs removing.
func planCleanup(targetPath, ref string) (plannedAction, error) {
	content, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return plannedAction{path: targetPath, ref: ref, kind: actionNone}, nil
	}
	if err != nil {
		return plannedAction{}, err
	}

	newContent, removed := withRefRemoved(string(content), ref)
	if !removed {
		return plannedAction{path: targetPath, ref: ref, kind: actionNone}, nil
	}
	if strings.TrimSpace(newContent) == "" {
		return plannedAction{path: targetPath, ref: ref, kind: actionDelete}, nil
	}
	return plannedAction{path: targetPath, ref: ref, kind: actionRemoveRef, content: []byte(newContent)}, nil
}

// applyAction writes a single planned action to disk.
func applyAction(a plannedAction) error {
	switch a.kind {
	case actionNone:
		return nil
	case actionCreate, actionUpdate, actionRemoveRef:
		return os.WriteFile(a.path, a.content, targetFileMode)
	case actionDelete:
		return os.Remove(a.path)
	}
	return fmt.Errorf("unknown action kind %d for %s", a.kind, a.path)
}

// withRefPrepended returns content with ref inserted at the top, separated from
// any existing content by a single blank line. If ref already appears anywhere
// in the file, content is returned unchanged (changed=false) so a reference the
// user moved lower is left untouched. Leading blank lines are dropped so empty
// lines do not accumulate.
func withRefPrepended(content, ref string) (string, bool) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == ref {
			return content, false
		}
	}

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

	return strings.Join(newLines, "\n"), true
}

// withRefRemoved strips every standalone occurrence of ref — a line that, trimmed
// of surrounding whitespace, equals ref exactly — wherever it appears (not only
// at the top), mirroring withRefPrepended's "present anywhere" rule. One blank
// line immediately following each removed reference is dropped too, so empty
// lines do not accumulate. Lines that merely contain ref as a substring (e.g.
// "See @AGENTS.md for details.") are left untouched. It returns (content, false)
// when no standalone reference line is present.
func withRefRemoved(content, ref string) (string, bool) {
	lines := strings.Split(content, "\n")

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
		return content, false
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

	return strings.Join(newLines, "\n"), true
}
