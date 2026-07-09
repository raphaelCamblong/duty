// Package board is the pure domain model of a BOARD.md file (spec §4).
// Every operation is line-surgical: it locates the target line, changes only
// that line, and leaves every other byte intact — a board is never re-rendered
// from a model. The package touches no filesystem: all functions take bytes
// and return bytes.
package board

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DefaultSection is the task section every board always has. It is the
// default target for new rows and the one section pruning never removes.
const DefaultSection = "Open tasks"

const (
	tableHeader    = "| Task | Title | Status |"
	tableSeparator = "|------|-------|--------|"
	boardsSection  = "Boards"
)

var (
	footerRe    = regexp.MustCompile(`^Completed tasks \((\d+)\) archived: \[archive/\]\(archive/\)\.$`)
	separatorRe = regexp.MustCompile(`^\|[-: |]+\|$`)
)

// Render returns a skeleton BOARD.md: H1 = title, the convention line, an
// empty "## Open tasks" table, and the zero-count archive footer.
func Render(title string) []byte {
	return []byte("# " + title + "\n" +
		"\n" +
		"Convention: [README.md](README.md). Workers update their row's status via the CLI.\n" +
		"Order top-to-bottom is the intended build order.\n" +
		"\n" +
		"## " + DefaultSection + "\n" +
		"\n" +
		tableHeader + "\n" +
		tableSeparator + "\n" +
		"\n" +
		"Completed tasks (0) archived: [archive/](archive/).\n")
}

// Title returns the board's H1 text, or "" when the board has no H1.
func Title(content []byte) string {
	for _, line := range splitLines(content) {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[len("# "):])
		}
	}
	return ""
}

// FindRow returns the table row whose task link targets filename — the
// |-prefixed line containing "(filename)" — and whether such a row exists.
func FindRow(content []byte, filename string) (string, bool) {
	lines := splitLines(content)
	i := rowIndex(lines, filename)
	if i < 0 {
		return "", false
	}
	return lines[i], true
}

// AddRow appends a task row "| [id](filename) | title | status |" to the
// named section's table, creating the section above the footer when absent.
func AddRow(content []byte, section, id, filename, title, status string) ([]byte, error) {
	row := "| [" + id + "](" + filename + ") | " + title + " | " + status + " |"
	lines, err := insertRow(splitLines(content), section, row)
	if err != nil {
		return nil, err
	}
	return joinLines(lines), nil
}

// SetRowStatus rewrites the status cell of the row targeting filename. Only
// that cell changes; every other cell keeps its exact spacing.
func SetRowStatus(content []byte, filename, status string) ([]byte, error) {
	lines := splitLines(content)
	i := rowIndex(lines, filename)
	if i < 0 {
		return nil, fmt.Errorf("no board row for %s", filename)
	}
	cells := strings.Split(lines[i], "|")
	if len(cells) < 3 {
		return nil, fmt.Errorf("malformed board row for %s", filename)
	}
	cells[len(cells)-2] = " " + status + " "
	lines[i] = strings.Join(cells, "|")
	return joinLines(lines), nil
}

// RowStatus returns the status cell of a table row as returned by FindRow,
// and whether row parsed as one (at least three "|"-separated cells).
func RowStatus(row string) (string, bool) {
	cells := strings.Split(row, "|")
	if len(cells) < 3 {
		return "", false
	}
	return strings.TrimSpace(cells[len(cells)-2]), true
}

// MoveRow relocates the row targeting filename to the end of the named
// section's table, creating the section above the footer when absent. The row
// line moves byte-identical. A section left empty is not removed here; callers
// compose with PruneEmptySections.
func MoveRow(content []byte, filename, section string) ([]byte, error) {
	lines := splitLines(content)
	i := rowIndex(lines, filename)
	if i < 0 {
		return nil, fmt.Errorf("no board row for %s", filename)
	}
	row := lines[i]
	lines, err := insertRow(append(lines[:i], lines[i+1:]...), section, row)
	if err != nil {
		return nil, err
	}
	return joinLines(lines), nil
}

// DropRow removes the row targeting filename.
func DropRow(content []byte, filename string) ([]byte, error) {
	lines := splitLines(content)
	i := rowIndex(lines, filename)
	if i < 0 {
		return nil, fmt.Errorf("no board row for %s", filename)
	}
	return joinLines(append(lines[:i], lines[i+1:]...)), nil
}

// PruneEmptySections removes every section whose body holds nothing but blank
// lines and empty table scaffolding (header + separator, no rows). The default
// "Open tasks" section is never removed, however empty; sections holding any
// prose, bullet, or row are kept untouched.
func PruneEmptySections(content []byte) []byte {
	lines := splitLines(content)
	for i := len(lines) - 1; i >= 0; i-- {
		if !strings.HasPrefix(lines[i], "## ") {
			continue
		}
		if strings.TrimSpace(lines[i][len("## "):]) == DefaultSection {
			continue
		}
		end := sectionEnd(lines, i)
		if !sectionEmpty(lines, i, end) {
			continue
		}
		lines = append(lines[:i], lines[end:]...)
	}
	return joinLines(lines)
}

// SetArchivedCount rewrites the number in the footer line
// "Completed tasks (N) archived: [archive/](archive/).".
func SetArchivedCount(content []byte, n int) ([]byte, error) {
	lines := splitLines(content)
	f := footerIndex(lines)
	if f < 0 {
		return nil, fmt.Errorf("board footer not found")
	}
	m := footerRe.FindStringSubmatchIndex(lines[f])
	lines[f] = lines[f][:m[2]] + strconv.Itoa(n) + lines[f][m[3]:]
	return joinLines(lines), nil
}

// AddBoardBullet appends the sub-board bullet "- [name/](name/BOARD.md) — title"
// to the "## Boards" section, creating the section before the first task
// section when absent. name is the sub-board folder name, without slash.
func AddBoardBullet(content []byte, name, title string) ([]byte, error) {
	bullet := "- [" + name + "/](" + name + "/BOARD.md) — " + title
	lines := splitLines(content)
	start, end, ok := sectionBounds(lines, boardsSection)
	if !ok {
		return createBoardsSection(lines, bullet)
	}
	at := start + 1
	for i := end - 1; i > start; i-- {
		if strings.HasPrefix(lines[i], "- ") {
			at = i + 1
			break
		}
	}
	if at == start+1 && at < end && lines[at] == "" {
		at++
	}
	return joinLines(insertAt(lines, at, bullet)), nil
}

// insertRow appends row to the named section's table, creating the section
// above the footer when it does not exist.
func insertRow(lines []string, section, row string) ([]string, error) {
	start, end, ok := sectionBounds(lines, section)
	if !ok {
		return createSection(lines, section, row)
	}
	if last := lastTableLine(lines, start, end); last >= 0 {
		return insertAt(lines, last+1, row), nil
	}
	at := start + 1
	if at < end && lines[at] == "" {
		at++
	}
	return insertAt(lines, at, tableHeader, tableSeparator, row), nil
}

// createSection inserts a new section holding row directly above the footer.
func createSection(lines []string, section, row string) ([]string, error) {
	f := footerIndex(lines)
	if f < 0 {
		return nil, fmt.Errorf("cannot create section %q: board footer not found", section)
	}
	block := []string{"## " + section, "", tableHeader, tableSeparator, row, ""}
	if f > 0 && lines[f-1] != "" {
		block = append([]string{""}, block...)
	}
	return insertAt(lines, f, block...), nil
}

// createBoardsSection inserts a "## Boards" section holding bullet before the
// first section heading (or above the footer when the board has no sections).
func createBoardsSection(lines []string, bullet string) ([]byte, error) {
	at := firstHeadingIndex(lines)
	if at < 0 {
		at = footerIndex(lines)
	}
	if at < 0 {
		return nil, fmt.Errorf("cannot create %s section: board has no sections and no footer", boardsSection)
	}
	block := []string{"## " + boardsSection, "", bullet, ""}
	if at > 0 && lines[at-1] != "" {
		block = append([]string{""}, block...)
	}
	return joinLines(insertAt(lines, at, block...)), nil
}

// sectionBounds locates "## <section>": start is the heading line index, end
// the index of the next heading, the footer, or len(lines).
func sectionBounds(lines []string, section string) (start, end int, ok bool) {
	heading := "## " + section
	for i, line := range lines {
		if strings.TrimRight(line, " \t") != heading {
			continue
		}
		return i, sectionEnd(lines, i), true
	}
	return 0, 0, false
}

// sectionEnd returns the index of the line ending the section that starts at
// start: the next heading, the footer, or len(lines).
func sectionEnd(lines []string, start int) int {
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || footerRe.MatchString(lines[i]) {
			return i
		}
	}
	return len(lines)
}

// sectionEmpty reports whether the section body in (start, end) holds nothing
// but blank lines and empty table scaffolding.
func sectionEmpty(lines []string, start, end int) bool {
	for i := start + 1; i < end; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == tableHeader || separatorRe.MatchString(line) {
			continue
		}
		return false
	}
	return true
}

// rowIndex returns the index of the |-prefixed line containing "(filename)",
// or -1 when no such row exists.
func rowIndex(lines []string, filename string) int {
	needle := "(" + filename + ")"
	for i, line := range lines {
		if strings.HasPrefix(line, "|") && strings.Contains(line, needle) {
			return i
		}
	}
	return -1
}

// lastTableLine returns the index of the last |-prefixed line strictly inside
// (start, end), or -1 when the section holds no table.
func lastTableLine(lines []string, start, end int) int {
	for i := end - 1; i > start; i-- {
		if strings.HasPrefix(lines[i], "|") {
			return i
		}
	}
	return -1
}

// firstHeadingIndex returns the index of the first "## " heading, or -1.
func firstHeadingIndex(lines []string) int {
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			return i
		}
	}
	return -1
}

// footerIndex returns the index of the archive footer line, or -1.
func footerIndex(lines []string) int {
	for i, line := range lines {
		if footerRe.MatchString(line) {
			return i
		}
	}
	return -1
}

// insertAt returns lines with insert placed before index i.
func insertAt(lines []string, i int, insert ...string) []string {
	out := make([]string, 0, len(lines)+len(insert))
	out = append(out, lines[:i]...)
	out = append(out, insert...)
	out = append(out, lines[i:]...)
	return out
}

// splitLines splits content on newlines; joinLines is its exact inverse, so
// edits that only insert, remove, or replace whole lines are byte-surgical.
func splitLines(content []byte) []string {
	return strings.Split(string(content), "\n")
}

// joinLines reassembles lines split by splitLines.
func joinLines(lines []string) []byte {
	return []byte(strings.Join(lines, "\n"))
}
