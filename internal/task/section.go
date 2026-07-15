package task

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// reportHeading is where a created section is inserted before, keeping the
// report last (spec §3): the report always trails the directive sections.
const reportHeading = "Report"

// Section returns the body of the "## <heading>" section — every byte after
// the heading line up to (but not including) the next "## " heading or the end
// of the file — and whether such a section exists. Heading matching ignores
// case and trailing whitespace; the heading line itself is never included.
func Section(content []byte, heading string) (body []byte, ok bool) {
	start, end, found := sectionBounds(content, heading)
	if !found {
		return nil, false
	}
	return content[start:end], true
}

// ReplaceSection replaces the body of the "## <heading>" section with body,
// leaving the heading line and every byte outside the section untouched. A
// missing section is created — inserted before "## Report", or appended at the
// end of the file when there is no report. An empty heading is rejected.
func ReplaceSection(content []byte, heading string, body []byte) ([]byte, error) {
	if strings.TrimSpace(heading) == "" {
		return nil, fmt.Errorf("empty section heading")
	}
	if start, end, ok := sectionBounds(content, heading); ok {
		region := sectionRegion(body, end < len(content))
		return splice(content, start, end, region), nil
	}
	return createSection(content, heading, body), nil
}

// ReplaceSections applies a bulk-edit payload — a sequence of "## <name>"
// blocks — onto content: each named section's body is replaced (a missing one
// created, per ReplaceSection), in payload order, with every byte outside the
// touched sections surviving. It errors when payload does not open at a "## "
// heading or names an empty section.
func ReplaceSections(content, payload []byte) ([]byte, error) {
	if err := RequireOpensAtSection(payload); err != nil {
		return nil, err
	}
	var err error
	for pos := nextHeadingFrom(payload, 0); pos < len(payload); {
		line, bodyStart := lineAt(payload, pos)
		end := nextHeadingFrom(payload, bodyStart)
		if content, err = ReplaceSection(content, headingName(line), payload[bodyStart:end]); err != nil {
			return nil, err
		}
		pos = end
	}
	return content, nil
}

// OpensAtSection reports whether content, after any leading blank lines, begins
// at a "## " section heading — the shape a task body and a bulk-set payload
// must have.
func OpensAtSection(content []byte) bool {
	at := nextHeadingFrom(content, 0)
	return at < len(content) && len(bytes.TrimSpace(content[:at])) == 0
}

// RequireOpensAtSection returns an error unless content opens at a "## " section
// heading (OpensAtSection): the one guard both a one-shot task body and a
// bulk-set payload are checked against.
func RequireOpensAtSection(content []byte) error {
	if !OpensAtSection(content) {
		return errors.New(`body must start at a "## " heading`)
	}
	return nil
}

// createSection inserts a fresh "## <heading>" section: before the report
// section when one exists, else appended at the end of the file.
func createSection(content []byte, heading string, body []byte) []byte {
	if at, ok := headingIndex(content, reportHeading); ok {
		block := append([]byte("## "+heading+"\n"), sectionRegion(body, true)...)
		return splice(content, at, at, block)
	}
	return appendSection(content, heading, body)
}

// appendSection writes a fresh "## <heading>" section at the end of content,
// blank-line separated from whatever precedes it.
func appendSection(content []byte, heading string, body []byte) []byte {
	var b bytes.Buffer
	writeEndingNL(&b, content)
	ensureBlankLine(&b)
	b.WriteString("## ")
	b.WriteString(heading)
	b.WriteByte('\n')
	b.Write(sectionRegion(body, false))
	return b.Bytes()
}

// sectionRegion renders a section body region — the bytes between a heading
// line and the next "## " heading or end of file — from text: the trimmed
// text, and a blank separator line when a heading follows. Empty text yields
// just that separator, matching the skeleton's blank sections.
func sectionRegion(text []byte, followed bool) []byte {
	var b bytes.Buffer
	if trimmed := bytes.TrimSpace(text); len(trimmed) > 0 {
		b.Write(trimmed)
		b.WriteByte('\n')
	}
	if followed {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// sectionBounds locates the "## <heading>" section body by byte offset: start
// just past the heading line, end at the next "## " heading or the end of the
// file. found is false when no heading matches.
func sectionBounds(content []byte, heading string) (start, end int, found bool) {
	at, ok := headingIndex(content, heading)
	if !ok {
		return 0, 0, false
	}
	_, start = lineAt(content, at)
	return start, nextHeadingFrom(content, start), true
}

// headingIndex returns the byte offset of the "## <heading>" line and whether
// it is present; matching ignores case and trailing whitespace.
func headingIndex(content []byte, heading string) (int, bool) {
	for pos := 0; pos < len(content); {
		line, next := lineAt(content, pos)
		if headingMatch(line, heading) {
			return pos, true
		}
		pos = next
	}
	return 0, false
}

// nextHeadingFrom returns the offset of the first "## " heading at or after
// pos, or len(content) when none follows.
func nextHeadingFrom(content []byte, pos int) int {
	for pos < len(content) {
		line, next := lineAt(content, pos)
		if isSectionHeading(line) {
			return pos
		}
		pos = next
	}
	return len(content)
}

// headingMatch reports whether line is a "## " heading whose text equals
// heading, ignoring case and trailing whitespace.
func headingMatch(line []byte, heading string) bool {
	return isSectionHeading(line) && strings.EqualFold(headingName(line), strings.TrimSpace(heading))
}

// headingName returns the text of a "## " heading line, trimmed of the marker
// and surrounding whitespace. It assumes line is a section heading.
func headingName(line []byte) string {
	rest, _ := strings.CutPrefix(strings.TrimRight(string(line), " \t\r"), "## ")
	return strings.TrimSpace(rest)
}

// isSectionHeading reports whether line opens a "## " level-two section.
func isSectionHeading(line []byte) bool {
	return bytes.HasPrefix(bytes.TrimRight(line, " \t\r"), []byte("## "))
}

// lineAt returns the line beginning at pos (without its newline) and the offset
// of the next line, len(content) when pos is on the final line.
func lineAt(content []byte, pos int) (line []byte, next int) {
	if i := bytes.IndexByte(content[pos:], '\n'); i >= 0 {
		return content[pos : pos+i], pos + i + 1
	}
	return content[pos:], len(content)
}

// splice returns content with the bytes in [start,end) replaced by mid.
func splice(content []byte, start, end int, mid []byte) []byte {
	out := make([]byte, 0, start+len(mid)+len(content)-end)
	out = append(out, content[:start]...)
	out = append(out, mid...)
	out = append(out, content[end:]...)
	return out
}
