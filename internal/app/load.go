package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// LoadOptions tunes what Load reads beyond the always-loaded open tree.
type LoadOptions struct {
	// Archive reads each archived file's contents into BoardView.Archived. When
	// false (the default) archives are only listed — their ids feed the dep
	// oracle, but their bytes are never read and Archived stays empty.
	Archive bool
}

// Load reads the whole duty tree above cwd into one TreeView: every board's
// index and open task files joined once, drift and waits computed from memory,
// archives listed (contents read only when opts.Archive). It is the single read
// path list/get/next/watch/tui project over. The whole tree loads regardless of
// scope — the dep oracle needs it — with scoping left to the projection.
func (a App) Load(cwd string, opts LoadOptions) (*TreeView, error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	contextDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return nil, err
	}
	dirs, err := tree.Boards(a.fs, root)
	if err != nil {
		return nil, err
	}
	view := &TreeView{
		Root: root, contextPath: relBoard(root, contextDir),
		byID:   map[string]*TaskView{},
		badIDs: map[string]bool{}, archivedIDs: map[string]bool{},
	}
	if err := a.loadBoards(view, dirs, opts); err != nil {
		return nil, err
	}
	view.index()
	view.annotate()
	return view, nil
}

func (a App) loadBoards(view *TreeView, dirs []string, opts LoadOptions) error {
	view.Boards = make([]BoardView, 0, len(dirs))
	for _, dir := range dirs {
		loaded, err := a.loadBoard(view, dir, opts)
		if err != nil {
			return err
		}
		view.Boards = append(view.Boards, loaded)
	}
	return nil
}

func (a App) loadBoard(view *TreeView, dir string, opts LoadOptions) (BoardView, error) {
	index, err := a.fs.ReadFile(boardIndexPath(dir))
	if err != nil {
		return BoardView{}, err
	}
	files, err := a.readBoardFiles(dir)
	if err != nil {
		return BoardView{}, err
	}
	loaded := BoardView{
		Dir:      dir,
		Path:     relBoard(view.Root, dir),
		Title:    board.TitleOr(index, filepath.Base(dir)),
		Sections: joinSections(dir, index, files),
	}
	if err := a.loadArchive(view, dir, &loaded, opts); err != nil {
		return BoardView{}, err
	}
	return loaded, nil
}

// fileEntry is one task file read from a board directory: view carries its
// parsed file truth when ok, raw holds the bytes when the frontmatter would
// not parse.
type fileEntry struct {
	view TaskView
	raw  []byte
	ok   bool
}

func (a App) readBoardFiles(dir string) (map[string]fileEntry, error) {
	entries, err := a.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	files := make(map[string]fileEntry)
	for _, entry := range entries {
		if entry.IsDir() || !tree.IsTaskFile(entry.Name()) {
			continue
		}
		loaded, err := a.readBoardFile(dir, entry)
		if err != nil {
			return nil, err
		}
		files[entry.Name()] = loaded
	}
	return files, nil
}

func (a App) readBoardFile(dir string, entry fs.DirEntry) (fileEntry, error) {
	path := filepath.Join(dir, entry.Name())
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return fileEntry{}, err
	}
	parsed, err := task.Parse(content)
	if err != nil {
		return fileEntry{raw: content}, nil
	}
	done, total := task.CountGates(content)
	return fileEntry{ok: true, view: TaskView{
		ID: parsed.ID, Title: parsed.Title, Status: parsed.Status,
		BlockedBy: parsed.BlockedBy, ClaimedBy: parsed.ClaimedBy,
		GatesDone: done, GatesTotal: total,
		Content: content, Path: path, File: entry.Name(),
		UpdatedAt: entryModTime(entry),
	}}, nil
}

// joinSections joins each board row to its task file in index order, then sorts
// every rowless file by name into the default section. A duplicate row for an
// already-joined file is dropped: the first row wins.
func joinSections(dir string, index []byte, files map[string]fileEntry) []SectionView {
	consumed := make(map[string]bool)
	var sections []SectionView
	for _, sec := range board.Sections(index) {
		sections = append(sections, joinSection(dir, sec, files, consumed))
	}
	return appendStrays(dir, sections, files, consumed)
}

// joinSection joins one board section's rows to their files in index order,
// marking each consumed so a later duplicate row (and the stray sweep) skips it.
func joinSection(dir string, sec board.Section, files map[string]fileEntry, consumed map[string]bool) SectionView {
	section := SectionView{Name: sec.Name}
	for _, row := range sec.Rows {
		if consumed[row.File] {
			continue
		}
		consumed[row.File] = true
		section.Tasks = append(section.Tasks, joinRow(dir, row, files))
	}
	return section
}

// joinRow merges one board row with its task file: file truth wins, the row
// only orders. A missing file, an unparsable one, or a status disagreement each
// becomes its own drift class.
func joinRow(dir string, row board.Row, files map[string]fileEntry) TaskView {
	entry, ok := files[row.File]
	if !ok {
		return TaskView{
			ID: row.ID, Title: row.Title, Status: row.Status,
			File: row.File, RowStatus: row.Status, Drift: DriftNoFile,
		}
	}
	if !entry.ok {
		return badFileTask(dir, row.File, entry.raw, row)
	}
	joined := entry.view
	if row.Status != joined.Status {
		joined.RowStatus = row.Status
		joined.Drift = DriftStatus
	}
	return joined
}

// badFileTask is an unparsable file's drift row: its identity comes from the
// board row when one indexes it, else from the filename; the raw bytes ride
// along so a formatter can still show the body.
func badFileTask(dir, name string, raw []byte, row board.Row) TaskView {
	return TaskView{
		ID: rowOr(row.ID, taskIDFromName(name)), Title: row.Title, Status: row.Status,
		File: name, Path: filepath.Join(dir, name),
		RowStatus: row.Status, Drift: DriftBadFile, Content: raw,
	}
}

// appendStrays sorts every rowless task file by name into the default section,
// appending that section last when the index has none.
func appendStrays(dir string, sections []SectionView, files map[string]fileEntry, consumed map[string]bool) []SectionView {
	strays := strayNames(files, consumed)
	if len(strays) == 0 {
		return sections
	}
	sections, at := ensureDefaultSection(sections)
	for _, name := range strays {
		sections[at].Tasks = append(sections[at].Tasks, strayTask(dir, name, files[name]))
	}
	return sections
}

func strayNames(files map[string]fileEntry, consumed map[string]bool) []string {
	var strays []string
	for name := range files {
		if !consumed[name] {
			strays = append(strays, name)
		}
	}
	sort.Strings(strays)
	return strays
}

func ensureDefaultSection(sections []SectionView) ([]SectionView, int) {
	for i := range sections {
		if sections[i].Name == board.DefaultSection {
			return sections, i
		}
	}
	return append(sections, SectionView{Name: board.DefaultSection}), len(sections)
}

func strayTask(dir, name string, entry fileEntry) TaskView {
	if !entry.ok {
		return badFileTask(dir, name, entry.raw, board.Row{})
	}
	stray := entry.view
	stray.Drift = DriftNoRow
	return stray
}

// loadArchive lists board dir's archive directory: it always counts the files
// and records their ids for the dep oracle, and reads their contents into
// Archived only when opts.Archive, so the toggle-off path reads no archived bytes.
func (a App) loadArchive(view *TreeView, dir string, loaded *BoardView, opts LoadOptions) error {
	archiveDir := filepath.Join(dir, names.ArchiveDir)
	entries, err := a.fs.ReadDir(archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", archiveDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !tree.IsTaskFile(entry.Name()) {
			continue
		}
		loaded.ArchivedCount++
		view.archivedIDs[taskIDFromName(entry.Name())] = true
		if !opts.Archive {
			continue
		}
		if err := a.appendArchived(loaded, archiveDir, entry); err != nil {
			return err
		}
	}
	return nil
}

func (a App) appendArchived(loaded *BoardView, archiveDir string, entry fs.DirEntry) error {
	archived, err := a.readArchived(archiveDir, entry)
	if err != nil {
		return err
	}
	loaded.Archived = append(loaded.Archived, archived)
	return nil
}

// readArchived reads one archived task file into a TaskView: file truth when it
// parses, id recovered from the filename when it does not.
func (a App) readArchived(dir string, entry fs.DirEntry) (TaskView, error) {
	path := filepath.Join(dir, entry.Name())
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return TaskView{}, err
	}
	archived := TaskView{File: entry.Name(), Path: path, Content: content, UpdatedAt: entryModTime(entry)}
	parsed, err := task.Parse(content)
	if err != nil {
		archived.ID = taskIDFromName(entry.Name())
		return archived, nil
	}
	archived.ID, archived.Title, archived.Status = parsed.ID, parsed.Title, parsed.Status
	archived.GatesDone, archived.GatesTotal = task.CountGates(content)
	return archived, nil
}

// index records every open file-truth task by id, so Task and the dep oracle
// answer from memory, and every bad-file id so a projection can name that fault.
// It runs after every board is built, so the pointers it keeps into the section
// slices stay valid.
func (view *TreeView) index() {
	for bi := range view.Boards {
		view.indexBoard(&view.Boards[bi])
	}
}

func (view *TreeView) indexBoard(boardView *BoardView) {
	for si := range boardView.Sections {
		view.indexSection(&boardView.Sections[si])
	}
}

func (view *TreeView) indexSection(section *SectionView) {
	for ti := range section.Tasks {
		item := &section.Tasks[ti]
		if item.Drift == DriftBadFile {
			view.badIDs[item.ID] = true
			continue
		}
		if item.Path == "" {
			continue
		}
		if _, seen := view.byID[item.ID]; !seen {
			view.byID[item.ID] = item
		}
	}
}

// annotate fills every open task's Deps and Waits from the dep oracle — map
// lookups over the loaded statuses, no further reads.
func (view *TreeView) annotate() {
	for _, item := range view.byID {
		item.Deps = view.deps(item.BlockedBy)
		item.Waits = unmetFromDeps(item.Deps)
	}
}

func (view *TreeView) deps(blockedBy []string) []Dep {
	deps := make([]Dep, 0, len(blockedBy))
	for _, id := range blockedBy {
		deps = append(deps, Dep{ID: id, Status: view.depStatus(id)})
	}
	return deps
}

func unmetFromDeps(deps []Dep) []string {
	var unmet []string
	for _, dep := range deps {
		if !depMet(dep.Status) {
			unmet = append(unmet, dep.ID)
		}
	}
	return unmet
}

// depStatus is the in-memory dep oracle: an open task answers its own status,
// an archived id answers "archived", anything else "missing". Board-only rows
// carry no file truth, so they never enter byID and never satisfy a dep.
func (view *TreeView) depStatus(id string) string {
	if open, ok := view.byID[id]; ok {
		return open.Status
	}
	if view.archivedIDs[id] {
		return statusArchived
	}
	return statusMissing
}

// taskIDFromName recovers a task id from a T-NN-<slug>.md filename — the label
// for an archived or unparsable file whose frontmatter cannot be read.
func taskIDFromName(name string) string {
	parts := strings.SplitN(name, "-", 3)
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return strings.TrimSuffix(name, ".md")
}

func rowOr(rowValue, fallback string) string {
	if rowValue != "" {
		return rowValue
	}
	return fallback
}

func entryModTime(entry fs.DirEntry) time.Time {
	info, err := entry.Info()
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
