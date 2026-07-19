// Package board is the pure domain model of a board index file (spec §4).
// Every operation is line-surgical: it locates the target line, changes only
// that line, and leaves every other byte intact — a board is never re-rendered
// from a model. The package touches no filesystem: all functions take bytes
// and return bytes.
package board

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/raphaelCamblong/duty/internal/names"
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
	footerRe    = regexp.MustCompile(`^Completed tasks \((\d+)\) archived: \[` + names.ArchiveDir + `/\]\(` + names.ArchiveDir + `/\)\.$`)
	separatorRe = regexp.MustCompile(`^\|[-: |]+\|$`)
)

//go:embed board.md.tmpl
var skeletonTmplText string

var skeletonTmpl = template.Must(template.New("board").Parse(skeletonTmplText))

// Render returns a skeleton board index: H1 = title, the convention line, an
// empty "## Open tasks" table, and the zero-count archive footer.
func Render(title string) []byte {
	var b bytes.Buffer
	_ = skeletonTmpl.Execute(&b, struct {
		Title, Readme, Section, Header, Separator, Archive string
	}{
		Title:     title,
		Readme:    names.ReadmeFile,
		Section:   DefaultSection,
		Header:    tableHeader,
		Separator: tableSeparator,
		Archive:   names.ArchiveDir,
	})
	return b.Bytes()
}

func Title(content []byte) string {
	for _, line := range splitLines(content) {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[len("# "):])
		}
	}
	return ""
}

func TitleOr(content []byte, fallback string) string {
	if t := Title(content); t != "" {
		return t
	}
	return fallback
}

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

func locateRow(content []byte, filename string) ([]string, int, error) {
	lines := splitLines(content)
	i := rowIndex(lines, filename)
	if i < 0 {
		return nil, 0, fmt.Errorf("no board row for %s", filename)
	}
	return lines, i, nil
}

func SetRowStatus(content []byte, filename, status string) ([]byte, error) {
	lines, i, err := locateRow(content, filename)
	if err != nil {
		return nil, err
	}
	cells := strings.Split(lines[i], "|")
	if len(cells) < 3 {
		return nil, fmt.Errorf("malformed board row for %s", filename)
	}
	cells[len(cells)-2] = " " + status + " "
	lines[i] = strings.Join(cells, "|")
	return joinLines(lines), nil
}

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
	lines, i, err := locateRow(content, filename)
	if err != nil {
		return nil, err
	}
	row := lines[i]
	lines, err = insertRow(append(lines[:i], lines[i+1:]...), section, row)
	if err != nil {
		return nil, err
	}
	return joinLines(lines), nil
}

// ReorderTop relocates the row targeting filename to the top of its section —
// above the section's first task row — preserving the row's bytes exactly. A
// row already at the top is left byte-identical.
func ReorderTop(content []byte, filename string) ([]byte, error) {
	lines, i, err := locateRow(content, filename)
	if err != nil {
		return nil, err
	}
	return joinLines(relocateRow(lines, i, sectionTop(lines, i))), nil
}

// ReorderBefore relocates the row targeting filename to sit immediately above
// the row targeting ref, preserving the moved row's bytes exactly. When ref is
// in another section the row adopts it — the move is purely positional.
func ReorderBefore(content []byte, filename, ref string) ([]byte, error) {
	return reorderAdjacent(content, filename, ref, 0)
}

// ReorderAfter relocates the row targeting filename to sit immediately below
// the row targeting ref, preserving the moved row's bytes exactly. When ref is
// in another section the row adopts it — the move is purely positional.
func ReorderAfter(content []byte, filename, ref string) ([]byte, error) {
	return reorderAdjacent(content, filename, ref, 1)
}

// reorderAdjacent relocates filename's row to ref's row index plus offset (0
// above ref, 1 below), preserving the moved row's bytes.
func reorderAdjacent(content []byte, filename, ref string, offset int) ([]byte, error) {
	lines, i, err := locateRow(content, filename)
	if err != nil {
		return nil, err
	}
	r := rowIndex(lines, ref)
	if r < 0 {
		return nil, fmt.Errorf("no board row for %s", ref)
	}
	return joinLines(relocateRow(lines, i, r+offset)), nil
}

func DropRow(content []byte, filename string) ([]byte, error) {
	lines, i, err := locateRow(content, filename)
	if err != nil {
		return nil, err
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
		if !isHeading(lines[i]) {
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

// AddBoardBullet appends a track bullet linking name/ to its board index,
// with title, to the "## Boards" section, creating the section before the
// first task section when absent. name is the track folder, without slash.
func AddBoardBullet(content []byte, name, title string) ([]byte, error) {
	bullet := "- [" + name + "/](" + name + "/" + names.BoardFile + ") — " + title
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

func sectionEnd(lines []string, start int) int {
	for i := start + 1; i < len(lines); i++ {
		if isHeading(lines[i]) || footerRe.MatchString(lines[i]) {
			return i
		}
	}
	return len(lines)
}

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

func rowIndex(lines []string, filename string) int {
	needle := "(" + filename + ")"
	for i, line := range lines {
		if strings.HasPrefix(line, "|") && strings.Contains(line, needle) {
			return i
		}
	}
	return -1
}

// relocateRow moves the line at from to sit before index at (both indices into
// the original slice), returning the reordered lines with the moved line's
// bytes intact. from == at leaves the slice unchanged.
func relocateRow(lines []string, from, at int) []string {
	row := lines[from]
	rest := append(lines[:from], lines[from+1:]...)
	if at > from {
		at--
	}
	return insertAt(rest, at, row)
}

// sectionTop returns the index of the first task row in the section containing
// line i — the insertion point that makes a row the section's first.
func sectionTop(lines []string, i int) int {
	start := sectionStart(lines, i)
	end := sectionEnd(lines, start)
	for j := start + 1; j < end; j++ {
		if rowLinkRe.MatchString(lines[j]) {
			return j
		}
	}
	return end
}

func sectionStart(lines []string, i int) int {
	for j := i; j >= 0; j-- {
		if isHeading(lines[j]) {
			return j
		}
	}
	return 0
}

func lastTableLine(lines []string, start, end int) int {
	for i := end - 1; i > start; i-- {
		if strings.HasPrefix(lines[i], "|") {
			return i
		}
	}
	return -1
}

func firstHeadingIndex(lines []string) int {
	for i, line := range lines {
		if isHeading(line) {
			return i
		}
	}
	return -1
}

func footerIndex(lines []string) int {
	for i, line := range lines {
		if footerRe.MatchString(line) {
			return i
		}
	}
	return -1
}

func insertAt(lines []string, i int, insert ...string) []string {
	out := make([]string, 0, len(lines)+len(insert))
	out = append(out, lines[:i]...)
	out = append(out, insert...)
	out = append(out, lines[i:]...)
	return out
}

func isHeading(line string) bool {
	return strings.HasPrefix(line, "## ")
}

// splitLines splits content on newlines; joinLines is its exact inverse, so
// edits that only insert, remove, or replace whole lines are byte-surgical.
func splitLines(content []byte) []string {
	return strings.Split(string(content), "\n")
}

func joinLines(lines []string) []byte {
	return []byte(strings.Join(lines, "\n"))
}
