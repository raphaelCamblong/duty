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
	Rows []Row
}

// Row is one task row of a section table: the id/file link plus the title and
// status cells. Sections parses it out of a board; AddRow writes one in.
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

// Sections parses the task sections of a board: every "## " heading except Boards,
// each with its table rows in order; prose, scaffolding, and the footer are skipped.
func Sections(content []byte) []Section {
	var out []Section
	open := false
	for _, line := range splitLines(content) {
		if isHeading(line) {
			out, open = startSection(out, line)
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

// startSection appends a task section for a "## " heading, skipping Boards; it
// reports whether a section is now open for rows.
func startSection(out []Section, line string) ([]Section, bool) {
	name := strings.TrimSpace(line[len("## "):])
	if name == boardsSection {
		return out, false
	}
	return append(out, Section{Name: name}), true
}

func parseRow(line string) (Row, bool) {
	match := rowLinkRe.FindStringSubmatch(line)
	if match == nil {
		return Row{}, false
	}
	cells := strings.Split(line, "|")
	if len(cells) < 5 {
		return Row{}, false
	}
	return Row{
		ID:     match[1],
		File:   match[2],
		Title:  strings.TrimSpace(cells[2]),
		Status: strings.TrimSpace(cells[len(cells)-2]),
	}, true
}
