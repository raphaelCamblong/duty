// Package tui is the live board viewer: a read-only Bubble Tea program that
// renders the tree from the task files (truth) in the order each board index
// gives, and re-scans on any filesystem event. It never writes.
package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Snapshot is the whole tree read at one instant: every board keyed by its
// slash-separated path relative to the root ("." for the root board).
type Snapshot struct {
	// Boards maps board path to its view model.
	Boards map[string]Board
}

// Board is one board's view model: identity, its direct sub-boards, and its
// task rows grouped in board-index section order.
type Board struct {
	// Path is the board's slash path relative to the root, "." for the root.
	Path string
	// Title is the board index H1, falling back to the folder name.
	Title string
	// Parent is the containing board's path, "" for the root.
	Parent string
	// Subs are the direct sub-boards, in lexical path order.
	Subs []Sub
	// Sections are the task sections in board-index order; task files with no
	// board row are appended to the default section with a drift flag.
	Sections []Section
	// Done and Total count open tasks read from the files — done vs all —
	// in this board and every board below it.
	Done, Total int
	// Counts tallies this subtree's open tasks by status (file truth).
	Counts map[string]int
}

// Sub is one sub-board line of the parent's view, counts live from files.
type Sub struct {
	// Path is the sub-board's path relative to the root.
	Path string
	// Name is the folder path relative to the parent, trailing slash.
	Name string
	// Title is the sub-board's H1.
	Title string
	// Done and Total count the sub-board's subtree like Board's.
	Done, Total int
	// Counts tallies the sub-board's subtree by status, like Board's.
	Counts map[string]int
}

// Section is one "## <name>" group of task rows.
type Section struct {
	// Name is the section heading text.
	Name string
	// Rows are the section's tasks in board order.
	Rows []Row
}

// Row is one task line: file truth (status, title, gates) joined with its
// board row; Drift names any disagreement between the two.
type Row struct {
	// ID, Title and Status come from the file when it exists, else from the
	// board row.
	ID, Title, Status string
	// File is the task filename; Path its absolute location, "" when the
	// board row points at a file that does not exist.
	File, Path string
	// GatesDone and GatesTotal count ticked vs all gate checkboxes.
	GatesDone, GatesTotal int
	// Goal is the task's "## Goal" section text, read once during the scan
	// so navigation previews it without reopening the file.
	Goal string
	// Drift is "" when file and board agree, else "board says <status>",
	// "no row", "no file", or "unparsable file".
	Drift string
	// Content is the raw task file, kept for the detail view.
	Content []byte
}

// Scan reads every board under root into a Snapshot. Archived tasks are
// invisible: board discovery skips archive/ directories.
func Scan(f fsys.FS, root string) (Snapshot, error) {
	dirs, err := tree.Boards(f, root)
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{Boards: make(map[string]Board, len(dirs))}
	paths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return Snapshot{}, fmt.Errorf("scan %s: %w", dir, err)
		}
		path := filepath.ToSlash(rel)
		b, err := scanBoard(f, dir, path)
		if err != nil {
			return Snapshot{}, err
		}
		snap.Boards[path] = b
		paths = append(paths, path)
	}
	link(snap, paths)
	return snap, nil
}

// scanBoard reads one board directory: its index for title and row order,
// its task files for truth. Done/Total are local here; link aggregates.
func scanBoard(f fsys.FS, dir, path string) (Board, error) {
	index, err := f.ReadFile(filepath.Join(dir, names.BoardFile))
	if err != nil {
		return Board{}, err
	}
	title := board.Title(index)
	if title == "" {
		title = filepath.Base(dir)
	}
	files, bad, err := readTasks(f, dir)
	if err != nil {
		return Board{}, err
	}

	b := Board{Path: path, Title: title}
	used := make(map[string]bool)
	for _, sec := range board.Sections(index) {
		s := Section{Name: sec.Name}
		for _, r := range sec.Rows {
			s.Rows = append(s.Rows, joinRow(dir, r, files, bad, used))
		}
		b.Sections = append(b.Sections, s)
	}
	appendOrphans(&b, files, used)
	b.Counts = make(map[string]int)
	for _, f := range files {
		if f.Status == task.StatusDone {
			b.Done++
		}
		b.Total++
		b.Counts[f.Status]++
	}
	return b, nil
}

// readTasks parses every task file directly in dir: files maps filename to
// its truth Row; bad holds the raw content of files whose frontmatter does
// not parse.
func readTasks(f fsys.FS, dir string) (files map[string]Row, bad map[string][]byte, err error) {
	entries, err := f.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", dir, err)
	}
	files = make(map[string]Row)
	bad = make(map[string][]byte)
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		content, err := f.ReadFile(abs)
		if err != nil {
			return nil, nil, err
		}
		t, err := task.Parse(content)
		if err != nil {
			bad[e.Name()] = content
			continue
		}
		gd, gt := task.CountGates(content)
		files[e.Name()] = Row{
			ID: t.ID, Title: t.Title, Status: t.Status,
			File: e.Name(), Path: abs,
			GatesDone: gd, GatesTotal: gt,
			Goal:    task.Section(content, "Goal"),
			Content: content,
		}
	}
	return files, bad, nil
}

// joinRow merges one board row with its task file: the file wins on status
// and title, the board only orders; any disagreement becomes Drift.
func joinRow(dir string, r board.Row, files map[string]Row, bad map[string][]byte, used map[string]bool) Row {
	if f, ok := files[r.File]; ok {
		used[r.File] = true
		if r.Status != f.Status {
			f.Drift = "board says " + r.Status
		}
		return f
	}
	if content, ok := bad[r.File]; ok {
		used[r.File] = true
		return Row{
			ID: r.ID, Title: r.Title, Status: r.Status,
			File: r.File, Path: filepath.Join(dir, r.File),
			Drift: "unparsable file", Content: content,
		}
	}
	return Row{ID: r.ID, Title: r.Title, Status: r.Status, File: r.File, Drift: "no file"}
}

// appendOrphans adds task files that have no board row to the end of the
// default section, flagged as drift.
func appendOrphans(b *Board, files map[string]Row, used map[string]bool) {
	var names []string
	for name := range files {
		if !used[name] {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return
	}
	sort.Strings(names)
	idx := -1
	for i, s := range b.Sections {
		if s.Name == board.DefaultSection {
			idx = i
			break
		}
	}
	if idx < 0 {
		b.Sections = append(b.Sections, Section{Name: board.DefaultSection})
		idx = len(b.Sections) - 1
	}
	for _, name := range names {
		r := files[name]
		r.Drift = "no row"
		b.Sections[idx].Rows = append(b.Sections[idx].Rows, r)
	}
}

// link resolves each board's parent, rolls local counts up into subtree
// counts, and fills every parent's Subs. paths is in lexical order, which
// Subs inherits.
func link(snap Snapshot, paths []string) {
	local := make(map[string][2]int, len(paths))
	localCounts := make(map[string]map[string]int, len(paths))
	for _, p := range paths {
		b := snap.Boards[p]
		local[p] = [2]int{b.Done, b.Total}
		localCounts[p] = b.Counts
	}
	for _, p := range paths {
		b := snap.Boards[p]
		b.Done, b.Total = 0, 0
		b.Counts = make(map[string]int)
		for _, q := range paths {
			if within(q, p) {
				b.Done += local[q][0]
				b.Total += local[q][1]
				for st, n := range localCounts[q] {
					b.Counts[st] += n
				}
			}
		}
		b.Parent = parentOf(snap, p)
		snap.Boards[p] = b
	}
	for _, q := range paths {
		if q == "." {
			continue
		}
		c := snap.Boards[q]
		p := snap.Boards[c.Parent]
		p.Subs = append(p.Subs, Sub{
			Path: q, Name: subName(c.Parent, q), Title: c.Title,
			Done: c.Done, Total: c.Total, Counts: c.Counts,
		})
		snap.Boards[c.Parent] = p
	}
}

// within reports whether board path q is p or lies below it.
func within(q, p string) bool {
	return p == "." || q == p || strings.HasPrefix(q, p+"/")
}

// parentOf returns the nearest ancestor board of p, "." when only the root
// contains it, "" for the root itself.
func parentOf(snap Snapshot, p string) string {
	if p == "." {
		return ""
	}
	segs := strings.Split(p, "/")
	for i := len(segs) - 1; i > 0; i-- {
		cand := strings.Join(segs[:i], "/")
		if _, ok := snap.Boards[cand]; ok {
			return cand
		}
	}
	return "."
}

// subName renders a sub-board's display name: its path relative to the
// parent board, with a trailing slash.
func subName(parent, q string) string {
	if parent != "." {
		q = strings.TrimPrefix(q, parent+"/")
	}
	return q + "/"
}
