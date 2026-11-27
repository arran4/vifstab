package rcs

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	rcslib "github.com/arran4/golang-rcs"
)

// Restore returns a verified revision without writing to any destination. An
// empty revision selects the head. It never guesses the identity of legacy files.
func Restore(source, revision string, config Config, fs FS) ([]byte, error) {
	if config.Dir == "" && !config.Local {
		return nil, fmt.Errorf("no RCS backup directory was selected")
	}
	if fs == nil {
		fs = DefaultFS
	}
	identity, path, err := historyIdentity(source, config.Dir, fs)
	if err != nil {
		return nil, err
	}
	if identity != "/etc/fstab" && !config.Local {
		return nil, fmt.Errorf("adjacent RCS history was not enabled")
	}
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read RCS history for recovery: %w", err)
	}
	f, err := rcslib.ParseFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RCS recovery history: %w", err)
	}
	provenance, err := parseProvenance(f, identity)
	if err != nil {
		return nil, err
	}
	if revision == "" {
		revision = f.Head
	}
	return restoreRevision(f, provenance, revision)
}

func revisionLines(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// RCS trunk deltas run backwards from the newest full text. Use a linear full
// replacement delta rather than quadratic LCS generation for large edits.
func linkPreviousRevision(f *rcslib.File, newest, previous []byte) error {
	var delta rcslib.EdDiff
	if lines := revisionLines(string(previous)); len(lines) > 0 {
		delta = append(delta, rcslib.Add{LineStart: 0, Lines: lines})
	}
	if lines := revisionLines(string(newest)); len(lines) > 0 {
		delta = append(delta, rcslib.Delete{1, len(lines)})
	}
	for _, content := range f.RevisionContents {
		if content.Revision == f.Head {
			content.Text = delta.String()
			return nil
		}
	}
	return fmt.Errorf("previous RCS head content is missing")
}

type revisionReader struct {
	lines []string
	next  int
}

func (r *revisionReader) ReadLine() (string, error) {
	if r.next == len(r.lines) {
		return "", io.EOF
	}
	line := r.lines[r.next]
	r.next++
	return line, nil
}

type revisionWriter struct{ lines []string }

func (w *revisionWriter) WriteLine(line string) error {
	w.lines = append(w.lines, line)
	return nil
}

func restoreRevision(f *rcslib.File, provenance *archiveProvenance, target string) ([]byte, error) {
	headers := map[string]*rcslib.RevisionHead{}
	contents := map[string]*rcslib.RevisionContent{}
	for _, header := range f.RevisionHeads {
		name := header.Revision.String()
		if _, exists := headers[name]; exists {
			return nil, fmt.Errorf("duplicate RCS revision header %q", name)
		}
		headers[name] = header
	}
	for _, content := range f.RevisionContents {
		if _, exists := contents[content.Revision]; exists {
			return nil, fmt.Errorf("duplicate RCS revision content %q", content.Revision)
		}
		contents[content.Revision] = content
	}
	if len(headers) != len(contents) || len(headers) != len(provenance.Revisions) {
		return nil, fmt.Errorf("RCS revision metadata is incomplete")
	}
	head, ok := contents[f.Head]
	if !ok {
		return nil, fmt.Errorf("RCS head content is missing")
	}
	current := head.Text
	seen := map[string]bool{}
	var result []byte
	found := false
	for revision := f.Head; revision != ""; {
		if seen[revision] {
			return nil, fmt.Errorf("RCS revision chain contains a loop")
		}
		seen[revision] = true
		metadata, ok := provenance.Revisions[revision]
		if !ok || contentIdentity([]byte(current)) != metadata {
			return nil, fmt.Errorf("RCS revision %q content integrity mismatch", revision)
		}
		if revision == target {
			result = append([]byte{}, current...)
			found = true
		}
		header, ok := headers[revision]
		if !ok {
			return nil, fmt.Errorf("RCS revision header %q is missing", revision)
		}
		revision = header.NextRevision.String()
		if revision == "" {
			break
		}
		delta, ok := contents[revision]
		if !ok {
			return nil, fmt.Errorf("RCS delta %q is missing", revision)
		}
		ed, err := parseReplacementDelta(delta.Text)
		if err != nil {
			return nil, fmt.Errorf("invalid RCS delta %q: %w", revision, err)
		}
		var writer revisionWriter
		if err := ed.Apply(&revisionReader{lines: revisionLines(current)}, &writer); err != nil {
			return nil, fmt.Errorf("failed to apply RCS delta %q: %w", revision, err)
		}
		current = strings.Join(writer.lines, "\n")
		// The dependency's Checkout inherits the newest text's final newline.
		// Restore the recorded ending at every hop (including empty revisions).
		if provenance.Revisions[revision].Newline {
			current += "\n"
		}
	}
	if len(seen) != len(headers) {
		return nil, fmt.Errorf("RCS history contains unreachable revisions")
	}
	if !found {
		return nil, fmt.Errorf("RCS revision %q was not found", target)
	}
	return result, nil
}

// Parse only the full-replacement scripts emitted above. Reading payload lines
// with Cut preserves CR and arbitrary bytes; the dependency's Scanner-based
// delta parser strips CR from CRLF payloads. RCS file parsing and application of
// the resulting edit commands still use the existing library.
func parseReplacementDelta(text string) (rcslib.EdDiff, error) {
	var commands rcslib.EdDiff
	for text != "" {
		line, rest, ok := strings.Cut(text, "\n")
		if !ok {
			return nil, fmt.Errorf("unterminated RCS replacement command")
		}
		text = rest
		op, countText, ok := strings.Cut(line, " ")
		count, err := strconv.Atoi(countText)
		if !ok || err != nil || count <= 0 {
			return nil, fmt.Errorf("invalid RCS replacement count")
		}
		switch op {
		case "d1":
			if text != "" {
				return nil, fmt.Errorf("unexpected data after RCS replacement deletion")
			}
			commands = append(commands, rcslib.Delete{1, count})
		case "a0":
			if len(commands) != 0 {
				return nil, fmt.Errorf("unexpected RCS replacement addition")
			}
			if count > len(text) { // every payload line needs at least one LF
				return nil, fmt.Errorf("truncated RCS replacement payload")
			}
			lines := make([]string, 0, count)
			for range count {
				line, rest, ok := strings.Cut(text, "\n")
				if !ok {
					return nil, fmt.Errorf("truncated RCS replacement payload")
				}
				lines = append(lines, line)
				text = rest
			}
			commands = append(commands, rcslib.Add{LineStart: 0, Lines: lines})
		default:
			return nil, fmt.Errorf("unsupported RCS replacement command")
		}
	}
	return commands, nil
}
