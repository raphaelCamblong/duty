package board

import (
	"regexp"
	"strings"
)

// Section is a read-only projection of one task section: its heading name
// and its table rows in board order.
type Section struct {
	// Name is the heading text after "## ".
	Name string
	// Rows are the section's task rows, top to bottom.
	Rows []Row
}

// Row is one parsed task row of a section table.
type Row struct {
	// ID is the link text of the Task cell.
	ID string
	// File is the link target of the Task cell.
	File string
	// Title is the Title cell, trimmed.
	Title string
	// Status is the Status cell, trimmed.
	Status string
}

var rowLinkRe = regexp.MustCompile(`^\|\s*\[([^\]]+)\]\(([^)]+)\)\s*\|`)

// Sections parses the task sections of a board: every "## " heading except
// Boards (which tooling never reads), each with its table rows in order.
// Prose, table scaffolding, and the archive footer are skipped, not modeled.
func Sections(content []byte) []Section {
	var out []Section
	open := false
	for _, line := range splitLines(content) {
		if isHeading(line) {
			name := strings.TrimSpace(line[len("## "):])
			open = name != boardsSection
			if open {
				out = append(out, Section{Name: name})
			}
			continue
		}
		if !open {
			continue
		}
		if footerRe.MatchString(line) {
			open = false
			continue
		}
		if row, ok := parseRow(line); ok {
			out[len(out)-1].Rows = append(out[len(out)-1].Rows, row)
		}
	}
	return out
}

// parseRow decodes one task row; ok is false for scaffolding and prose.
func parseRow(line string) (Row, bool) {
	m := rowLinkRe.FindStringSubmatch(line)
	if m == nil {
		return Row{}, false
	}
	cells := strings.Split(line, "|")
	if len(cells) < 5 {
		return Row{}, false
	}
	return Row{
		ID:     m[1],
		File:   m[2],
		Title:  strings.TrimSpace(cells[2]),
		Status: strings.TrimSpace(cells[len(cells)-2]),
	}, true
}
