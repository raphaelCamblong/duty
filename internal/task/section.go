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

// Section returns the body of the "## <heading>" section — bytes after the heading
// line up to the next "## " heading or end of file — and whether it exists.
func Section(content []byte, heading string) (body []byte, ok bool) {
	start, end, found := sectionBounds(content, heading)
	if !found {
		return nil, false
	}
	return content[start:end], true
}

// ReplaceSection replaces the "## <heading>" section body, every byte outside it
// untouched; a missing section is created before "## Report", else at end of file.
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

// ReplaceSections applies a payload of "## <name>" blocks onto content in order,
// each section's body replaced per ReplaceSection, every other byte surviving.
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

// OpensAtSection reports whether content, after leading blank lines, begins at a
// "## " heading — the shape a task body and a bulk-set payload must have.
func OpensAtSection(content []byte) bool {
	at := nextHeadingFrom(content, 0)
	return at < len(content) && len(bytes.TrimSpace(content[:at])) == 0
}

// RequireOpensAtSection errors unless content OpensAtSection.
func RequireOpensAtSection(content []byte) error {
	if !OpensAtSection(content) {
		return errors.New(`body must start at a "## " heading`)
	}
	return nil
}

func createSection(content []byte, heading string, body []byte) []byte {
	if at, ok := headingIndex(content, reportHeading); ok {
		block := append([]byte("## "+heading+"\n"), sectionRegion(body, true)...)
		return splice(content, at, at, block)
	}
	return appendSection(content, heading, body)
}

func appendSection(content []byte, heading string, body []byte) []byte {
	var buf bytes.Buffer
	writeEndingNL(&buf, content)
	ensureBlankLine(&buf)
	buf.WriteString("## ")
	buf.WriteString(heading)
	buf.WriteByte('\n')
	buf.Write(sectionRegion(body, false))
	return buf.Bytes()
}

// sectionRegion renders a section body from text — the trimmed text plus a
// trailing blank line when followed is true — matching the skeleton's blanks.
func sectionRegion(text []byte, followed bool) []byte {
	var buf bytes.Buffer
	if trimmed := bytes.TrimSpace(text); len(trimmed) > 0 {
		buf.Write(trimmed)
		buf.WriteByte('\n')
	}
	if followed {
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func sectionBounds(content []byte, heading string) (start, end int, found bool) {
	at, ok := headingIndex(content, heading)
	if !ok {
		return 0, 0, false
	}
	_, start = lineAt(content, at)
	return start, nextHeadingFrom(content, start), true
}

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

func headingMatch(line []byte, heading string) bool {
	return isSectionHeading(line) && strings.EqualFold(headingName(line), strings.TrimSpace(heading))
}

// headingName returns the text of a "## " heading line, trimmed of the marker
// and surrounding whitespace. It assumes line is a section heading.
func headingName(line []byte) string {
	rest, _ := strings.CutPrefix(strings.TrimRight(string(line), " \t\r"), "## ")
	return strings.TrimSpace(rest)
}

func isSectionHeading(line []byte) bool {
	return bytes.HasPrefix(bytes.TrimRight(line, " \t\r"), []byte("## "))
}

func lineAt(content []byte, pos int) (line []byte, next int) {
	if nl := bytes.IndexByte(content[pos:], '\n'); nl >= 0 {
		return content[pos : pos+nl], pos + nl + 1
	}
	return content[pos:], len(content)
}

func splice(content []byte, start, end int, mid []byte) []byte {
	out := make([]byte, 0, start+len(mid)+len(content)-end)
	out = append(out, content[:start]...)
	out = append(out, mid...)
	out = append(out, content[end:]...)
	return out
}
