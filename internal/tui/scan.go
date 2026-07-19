// Package tui is duty's live board viewer.
package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Snapshot is the whole tree read at one instant: every board keyed by its
// slash-separated path relative to the root ("." for the root board).
type Snapshot struct {
	Boards map[string]Board
}

// anyInProgress reports whether the whole tree holds at least one in-progress
// task; the root board's rolled-up counts already tally every board below it,
// so this is one map lookup — a snapshot-level answer, not a per-row scan.
func (s Snapshot) anyInProgress() bool {
	return s.Boards["."].Counts[task.StatusInProgress] > 0
}

// Board is one board's view model: identity, its direct tracks, and its
// task rows grouped in board-index section order.
type Board struct {
	Path string
	// Title is the board index H1, falling back to the folder name.
	Title string
	// Parent is the containing board's path, "" for the root.
	Parent string
	// Subs are the direct tracks, in lexical path order.
	Subs []Sub
	// Sections are the task sections in board-index order; task files with no
	// board row are appended to the default section with a drift flag.
	Sections []Section
	// Done and Total count open tasks read from the files — done vs all —
	// in this board and every board below it.
	Done, Total int
	// Counts tallies this subtree's open tasks by status (file truth).
	Counts map[string]int
	// ArchivedCount is the number of archived task files in this board's own
	// archive/ directory — a cheap listing tallied on every scan.
	ArchivedCount int
	// ArchivedSubtree tallies archived tasks in this board and every board
	// below it; link fills it from each board's local ArchivedCount.
	ArchivedSubtree int
	// Archived holds this board's archived task rows, read from archive/ only
	// when the scan is asked to include them (the TUI's archive toggle); nil
	// otherwise. The dim archived list shows id, title, and age; the normal
	// read-only preview renders each file's body.
	Archived []Row
}

// Sub is one track line of the parent's view, counts live from files.
type Sub struct {
	Path string
	// Name is the folder path relative to the parent, trailing slash.
	Name  string
	Title string
	// Done and Total count the track's subtree like Board's.
	Done, Total int
	// Counts tallies the track's subtree by status, like Board's.
	Counts map[string]int
	// Archived tallies archived tasks across the track's subtree; the OFF
	// hiding rule reads it, the ON view shows it beside the reappeared track.
	Archived int
}

// Section is one "## <name>" group of task rows.
type Section struct {
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
	File, Path            string
	GatesDone, GatesTotal int
	BlockedBy             []string
	// Waits lists the BlockedBy ids not yet met — the wait state the scan
	// computes from the snapshot's own statuses; empty when the task is
	// actionable.
	Waits []string
	// ClaimedBy names the agent holding an in-progress task, "" when unclaimed.
	ClaimedBy string
	// Drift is "" when file and board agree, else "board says <status>",
	// "no row", "no file", or "unparsable file".
	Drift   string
	Content []byte
	// UpdatedAt is the task file's modification time, zero when it has no file.
	UpdatedAt time.Time
}

// Scan reads every board under root into a Snapshot. Each board's archived
// task count is always tallied from a cheap archive/ listing; the archived
// rows themselves are read only when includeArchive is set (the TUI's archive
// toggle), so the default path never reads an archived file's bytes.
func Scan(f fsys.FS, root string, includeArchive bool) (Snapshot, error) {
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
		b, err := scanBoard(f, dir, path, includeArchive)
		if err != nil {
			return Snapshot{}, err
		}
		snap.Boards[path] = b
		paths = append(paths, path)
	}
	link(snap, paths)
	annotateWaits(snap)
	return snap, nil
}

// annotateWaits fills each task row's Waits from the snapshot's own statuses:
// the blocked-by ids not yet met. It reads no files — done and archived deps
// have left the open set, so a dep absent from the snapshot counts as met,
// keeping this pass O(rows) so startup stays fast. A blocked-by that resolves
// nowhere is thus treated as met here (unlike the CLI, which can see archived
// files and flags a truly-missing id); the common archived case matches.
func annotateWaits(snap Snapshot) {
	status := make(map[string]string)
	for _, b := range snap.Boards {
		for _, s := range b.Sections {
			for _, r := range s.Rows {
				if r.ID != "" {
					status[r.ID] = r.Status
				}
			}
		}
	}
	statusOf := func(id string) (string, error) {
		if s, ok := status[id]; ok {
			return s, nil
		}
		return task.StatusDone, nil
	}
	for _, b := range snap.Boards {
		for si := range b.Sections {
			for ri := range b.Sections[si].Rows {
				r := &b.Sections[si].Rows[ri]
				r.Waits, _ = app.UnmetDeps(r.BlockedBy, statusOf)
			}
		}
	}
}

// scanBoard reads one board directory: its index for title and row order,
// its task files for truth, and its archive/ for the archived tally (rows too
// when includeArchive). Done/Total are local here; link aggregates.
func scanBoard(f fsys.FS, dir, path string, includeArchive bool) (Board, error) {
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
	bf := boardFiles{files: files, bad: bad, used: make(map[string]bool)}
	for _, sec := range board.Sections(index) {
		s := Section{Name: sec.Name}
		for _, r := range sec.Rows {
			s.Rows = append(s.Rows, joinRow(dir, r, bf))
		}
		b.Sections = append(b.Sections, s)
	}
	appendOrphans(&b, bf)
	tallyOpen(&b, files)
	b.ArchivedCount, b.Archived, err = scanArchive(f, filepath.Join(dir, names.ArchiveDir), includeArchive)
	if err != nil {
		return Board{}, err
	}
	return b, nil
}

// tallyOpen fills a board's local Done, Total, and per-status Counts from its
// open task files (file truth); link rolls these up into subtree totals.
func tallyOpen(b *Board, files map[string]Row) {
	b.Counts = make(map[string]int)
	for _, f := range files {
		if f.Status == task.StatusDone {
			b.Done++
		}
		b.Total++
		b.Counts[f.Status]++
	}
}

// scanArchive tallies a board's archive/ directory: the count is always the
// number of archived task files (a cheap listing), while rows are read from
// those files only when includeContent is set, so the toggle-off path never
// reads an archived file's bytes. A missing archive/ is simply no archive.
func scanArchive(f fsys.FS, archiveDir string, includeContent bool) (int, []Row, error) {
	entries, err := f.ReadDir(archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("scan archive %s: %w", archiveDir, err)
	}
	count := 0
	var rows []Row
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		count++
		if !includeContent {
			continue
		}
		row, err := readArchivedTask(f, archiveDir, e)
		if err != nil {
			return 0, nil, err
		}
		rows = append(rows, row)
	}
	return count, rows, nil
}

func readArchivedTask(f fsys.FS, dir string, e fs.DirEntry) (Row, error) {
	abs := filepath.Join(dir, e.Name())
	content, err := f.ReadFile(abs)
	if err != nil {
		return Row{}, err
	}
	r := Row{File: e.Name(), Path: abs, Content: content, UpdatedAt: entryModTime(e)}
	t, err := task.Parse(content)
	if err != nil {
		r.ID = archivedID(e.Name())
		return r, nil
	}
	r.ID, r.Title, r.Status = t.ID, t.Title, t.Status
	r.GatesDone, r.GatesTotal = task.CountGates(content)
	return r, nil
}

// archivedID recovers a task id from a T-NN-<slug>.md filename, the fallback
// label when an archived file's frontmatter will not parse.
func archivedID(name string) string {
	parts := strings.SplitN(name, "-", 3)
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return strings.TrimSuffix(name, ".md")
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
			BlockedBy: t.BlockedBy,
			ClaimedBy: t.ClaimedBy,
			Content:   content,
			UpdatedAt: entryModTime(e),
		}
	}
	return files, bad, nil
}

func entryModTime(e fs.DirEntry) time.Time {
	info, err := e.Info()
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// boardFiles indexes a board directory's parsed task files for joining with
// board rows: files by name, bad the raw bytes of unparsable ones, and used
// the set consumed by rows so appendOrphans can find the rest.
type boardFiles struct {
	files map[string]Row
	bad   map[string][]byte
	used  map[string]bool
}

// joinRow merges one board row with its task file: the file wins on status
// and title, the board only orders; any disagreement becomes Drift.
func joinRow(dir string, r board.Row, bf boardFiles) Row {
	if f, ok := bf.files[r.File]; ok {
		bf.used[r.File] = true
		if r.Status != f.Status {
			f.Drift = "board says " + r.Status
		}
		return f
	}
	if content, ok := bf.bad[r.File]; ok {
		bf.used[r.File] = true
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
func appendOrphans(b *Board, bf boardFiles) {
	var names []string
	for name := range bf.files {
		if !bf.used[name] {
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
		r := bf.files[name]
		r.Drift = "no row"
		b.Sections[idx].Rows = append(b.Sections[idx].Rows, r)
	}
}

// localAgg snapshots one board's own tallies before link rolls each board's
// subtree up from its descendants'.
type localAgg struct {
	done, total, arch int
	counts            map[string]int
}

// link resolves each board's parent, rolls local counts up into subtree
// counts, and fills every parent's Subs. paths is in lexical order, which
// Subs inherits.
func link(snap Snapshot, paths []string) {
	locals := make(map[string]localAgg, len(paths))
	for _, p := range paths {
		b := snap.Boards[p]
		locals[p] = localAgg{done: b.Done, total: b.Total, arch: b.ArchivedCount, counts: b.Counts}
	}
	for _, p := range paths {
		b := snap.Boards[p]
		b.Done, b.Total, b.ArchivedSubtree = 0, 0, 0
		b.Counts = make(map[string]int)
		for _, q := range paths {
			if !within(q, p) {
				continue
			}
			lq := locals[q]
			b.Done += lq.done
			b.Total += lq.total
			b.ArchivedSubtree += lq.arch
			for st, n := range lq.counts {
				b.Counts[st] += n
			}
		}
		b.Parent = parentOf(snap, p)
		snap.Boards[p] = b
	}
	buildSubs(snap, paths)
}

func buildSubs(snap Snapshot, paths []string) {
	for _, q := range paths {
		if q == "." {
			continue
		}
		c := snap.Boards[q]
		p := snap.Boards[c.Parent]
		p.Subs = append(p.Subs, Sub{
			Path: q, Name: subName(c.Parent, q), Title: c.Title,
			Done: c.Done, Total: c.Total, Counts: c.Counts,
			Archived: c.ArchivedSubtree,
		})
		snap.Boards[c.Parent] = p
	}
}

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

func subName(parent, q string) string {
	if parent != "." {
		q = strings.TrimPrefix(q, parent+"/")
	}
	return q + "/"
}
