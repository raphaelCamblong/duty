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
func Scan(filesystem fsys.FS, root string, includeArchive bool) (Snapshot, error) {
	dirs, err := tree.Boards(filesystem, root)
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
		scanned, err := scanBoard(filesystem, dir, path, includeArchive)
		if err != nil {
			return Snapshot{}, err
		}
		snap.Boards[path] = scanned
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
	for _, current := range snap.Boards {
		for _, section := range current.Sections {
			for _, row := range section.Rows {
				if row.ID != "" {
					status[row.ID] = row.Status
				}
			}
		}
	}
	statusOf := func(id string) (string, error) {
		if value, ok := status[id]; ok {
			return value, nil
		}
		return task.StatusDone, nil
	}
	for _, current := range snap.Boards {
		for si := range current.Sections {
			for ri := range current.Sections[si].Rows {
				row := &current.Sections[si].Rows[ri]
				row.Waits, _ = app.UnmetDeps(row.BlockedBy, statusOf)
			}
		}
	}
}

// scanBoard reads one board directory: its index for title and row order,
// its task files for truth, and its archive/ for the archived tally (rows too
// when includeArchive). Done/Total are local here; link aggregates.
func scanBoard(filesystem fsys.FS, dir, path string, includeArchive bool) (Board, error) {
	index, err := filesystem.ReadFile(filepath.Join(dir, names.BoardFile))
	if err != nil {
		return Board{}, err
	}
	title := board.Title(index)
	if title == "" {
		title = filepath.Base(dir)
	}
	files, bad, err := readTasks(filesystem, dir)
	if err != nil {
		return Board{}, err
	}

	result := Board{Path: path, Title: title}
	bf := boardFiles{files: files, bad: bad, used: make(map[string]bool)}
	for _, sec := range board.Sections(index) {
		section := Section{Name: sec.Name}
		for _, boardRow := range sec.Rows {
			section.Rows = append(section.Rows, joinRow(dir, boardRow, bf))
		}
		result.Sections = append(result.Sections, section)
	}
	appendOrphans(&result, bf)
	tallyOpen(&result, files)
	result.ArchivedCount, result.Archived, err = scanArchive(filesystem, filepath.Join(dir, names.ArchiveDir), includeArchive)
	if err != nil {
		return Board{}, err
	}
	return result, nil
}

// tallyOpen fills a board's local Done, Total, and per-status Counts from its
// open task files (file truth); link rolls these up into subtree totals.
func tallyOpen(target *Board, files map[string]Row) {
	target.Counts = make(map[string]int)
	for _, file := range files {
		if file.Status == task.StatusDone {
			target.Done++
		}
		target.Total++
		target.Counts[file.Status]++
	}
}

// scanArchive tallies a board's archive/ directory: the count is always the
// number of archived task files (a cheap listing), while rows are read from
// those files only when includeContent is set, so the toggle-off path never
// reads an archived file's bytes. A missing archive/ is simply no archive.
func scanArchive(filesystem fsys.FS, archiveDir string, includeContent bool) (int, []Row, error) {
	entries, err := filesystem.ReadDir(archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("scan archive %s: %w", archiveDir, err)
	}
	count := 0
	var rows []Row
	for _, dirEntry := range entries {
		if dirEntry.IsDir() || !tree.IsTaskFile(dirEntry.Name()) {
			continue
		}
		count++
		if !includeContent {
			continue
		}
		row, err := readArchivedTask(filesystem, archiveDir, dirEntry)
		if err != nil {
			return 0, nil, err
		}
		rows = append(rows, row)
	}
	return count, rows, nil
}

func readArchivedTask(filesystem fsys.FS, dir string, dirEntry fs.DirEntry) (Row, error) {
	abs := filepath.Join(dir, dirEntry.Name())
	content, err := filesystem.ReadFile(abs)
	if err != nil {
		return Row{}, err
	}
	row := Row{File: dirEntry.Name(), Path: abs, Content: content, UpdatedAt: entryModTime(dirEntry)}
	parsed, err := task.Parse(content)
	if err != nil {
		row.ID = archivedID(dirEntry.Name())
		return row, nil
	}
	row.ID, row.Title, row.Status = parsed.ID, parsed.Title, parsed.Status
	row.GatesDone, row.GatesTotal = task.CountGates(content)
	return row, nil
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
func readTasks(filesystem fsys.FS, dir string) (files map[string]Row, bad map[string][]byte, err error) {
	entries, err := filesystem.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", dir, err)
	}
	files = make(map[string]Row)
	bad = make(map[string][]byte)
	for _, dirEntry := range entries {
		if dirEntry.IsDir() || !tree.IsTaskFile(dirEntry.Name()) {
			continue
		}
		abs := filepath.Join(dir, dirEntry.Name())
		content, err := filesystem.ReadFile(abs)
		if err != nil {
			return nil, nil, err
		}
		parsed, err := task.Parse(content)
		if err != nil {
			bad[dirEntry.Name()] = content
			continue
		}
		gd, gt := task.CountGates(content)
		files[dirEntry.Name()] = Row{
			ID: parsed.ID, Title: parsed.Title, Status: parsed.Status,
			File: dirEntry.Name(), Path: abs,
			GatesDone: gd, GatesTotal: gt,
			BlockedBy: parsed.BlockedBy,
			ClaimedBy: parsed.ClaimedBy,
			Content:   content,
			UpdatedAt: entryModTime(dirEntry),
		}
	}
	return files, bad, nil
}

func entryModTime(dirEntry fs.DirEntry) time.Time {
	info, err := dirEntry.Info()
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
func joinRow(dir string, boardRow board.Row, bf boardFiles) Row {
	if fileRow, ok := bf.files[boardRow.File]; ok {
		bf.used[boardRow.File] = true
		if boardRow.Status != fileRow.Status {
			fileRow.Drift = "board says " + boardRow.Status
		}
		return fileRow
	}
	if content, ok := bf.bad[boardRow.File]; ok {
		bf.used[boardRow.File] = true
		return Row{
			ID: boardRow.ID, Title: boardRow.Title, Status: boardRow.Status,
			File: boardRow.File, Path: filepath.Join(dir, boardRow.File),
			Drift: "unparsable file", Content: content,
		}
	}
	return Row{ID: boardRow.ID, Title: boardRow.Title, Status: boardRow.Status, File: boardRow.File, Drift: "no file"}
}

// appendOrphans adds task files that have no board row to the end of the
// default section, flagged as drift.
func appendOrphans(target *Board, bf boardFiles) {
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
	for i, section := range target.Sections {
		if section.Name == board.DefaultSection {
			idx = i
			break
		}
	}
	if idx < 0 {
		target.Sections = append(target.Sections, Section{Name: board.DefaultSection})
		idx = len(target.Sections) - 1
	}
	for _, name := range names {
		row := bf.files[name]
		row.Drift = "no row"
		target.Sections[idx].Rows = append(target.Sections[idx].Rows, row)
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
	for _, path := range paths {
		current := snap.Boards[path]
		locals[path] = localAgg{done: current.Done, total: current.Total, arch: current.ArchivedCount, counts: current.Counts}
	}
	for _, path := range paths {
		current := snap.Boards[path]
		current.Done, current.Total, current.ArchivedSubtree = 0, 0, 0
		current.Counts = make(map[string]int)
		for _, candidate := range paths {
			if !within(candidate, path) {
				continue
			}
			lq := locals[candidate]
			current.Done += lq.done
			current.Total += lq.total
			current.ArchivedSubtree += lq.arch
			for st, count := range lq.counts {
				current.Counts[st] += count
			}
		}
		current.Parent = parentOf(snap, path)
		snap.Boards[path] = current
	}
	buildSubs(snap, paths)
}

func buildSubs(snap Snapshot, paths []string) {
	for _, path := range paths {
		if path == "." {
			continue
		}
		child := snap.Boards[path]
		parent := snap.Boards[child.Parent]
		parent.Subs = append(parent.Subs, Sub{
			Path: path, Name: subName(child.Parent, path), Title: child.Title,
			Done: child.Done, Total: child.Total, Counts: child.Counts,
			Archived: child.ArchivedSubtree,
		})
		snap.Boards[child.Parent] = parent
	}
}

func within(path, prefix string) bool {
	return prefix == "." || path == prefix || strings.HasPrefix(path, prefix+"/")
}

// parentOf returns the nearest ancestor board of p, "." when only the root
// contains it, "" for the root itself.
func parentOf(snap Snapshot, path string) string {
	if path == "." {
		return ""
	}
	segs := strings.Split(path, "/")
	for i := len(segs) - 1; i > 0; i-- {
		cand := strings.Join(segs[:i], "/")
		if _, ok := snap.Boards[cand]; ok {
			return cand
		}
	}
	return "."
}

func subName(parent, path string) string {
	if parent != "." {
		path = strings.TrimPrefix(path, parent+"/")
	}
	return path + "/"
}
