package tui

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raphaelCamblong/duty/internal/config"
)

// refreshMsg reports a debounced filesystem change from the watcher.
type refreshMsg struct{}

// snapMsg carries a completed re-scan.
type snapMsg struct {
	snap Snapshot
	err  error
}

// editedMsg reports that the $EDITOR process finished.
type editedMsg struct{ err error }

// Model is the Bubble Tea model of the viewer: one board on screen at a
// time, optionally a task detail on top. Update is a pure transition —
// filesystem reads happen in commands, writes nowhere.
type Model struct {
	root    string
	editor  string
	theme   string
	keys    keyMap
	refresh <-chan struct{}

	snap    Snapshot
	scanErr string
	path    string
	cursors map[string]int
	width   int
	height  int

	detailOpen bool
	detailRow  Row
	detailVP   viewport.Model
}

// New scans the tree under root and returns a model opened on the root
// board, styled per cfg.
func New(root string, cfg config.Config) (Model, error) {
	snap, err := Scan(root)
	if err != nil {
		return Model{}, err
	}
	return Model{
		root:    root,
		editor:  cfg.Editor,
		theme:   cfg.TUI.Theme,
		keys:    defaultKeys(),
		snap:    snap,
		path:    ".",
		cursors: map[string]int{},
	}, nil
}

// BoardPath returns the path of the board on screen ("." = root).
func (m Model) BoardPath() string { return m.path }

// Cursor returns the selected item index in the board view.
func (m Model) Cursor() int { return m.cursor() }

// DetailID returns the id of the task open in the detail view, "" if none.
func (m Model) DetailID() string {
	if !m.detailOpen {
		return ""
	}
	return m.detailRow.ID
}

// Init starts waiting on the watcher when one is attached.
func (m Model) Init() tea.Cmd {
	if m.refresh == nil {
		return nil
	}
	return waitRefresh(m.refresh)
}

// Update is the state transition for every message the program receives.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.detailOpen {
			m = m.openDetail(m.detailRow, m.detailVP.YOffset)
		}
		return m, nil
	case refreshMsg:
		if m.refresh == nil {
			return m, scanCmd(m.root)
		}
		return m, tea.Batch(scanCmd(m.root), waitRefresh(m.refresh))
	case snapMsg:
		return m.withSnap(msg), nil
	case editedMsg:
		return m, scanCmd(m.root)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey routes a key press: quit anywhere, detail keys when a task is
// open, board navigation otherwise.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}
	if m.detailOpen {
		return m.detailKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Down):
		return m.moveCursor(1), nil
	case key.Matches(msg, m.keys.Up):
		return m.moveCursor(-1), nil
	case key.Matches(msg, m.keys.Open):
		return m.open(), nil
	case key.Matches(msg, m.keys.Back):
		return m.back(), nil
	case key.Matches(msg, m.keys.Edit):
		if it, ok := m.selected(); ok && it.row != nil {
			return m, editCmd(m.editor, it.row.Path)
		}
	}
	return m, nil
}

// detailKey handles keys with the detail open: back closes, edit opens the
// task in $EDITOR, everything else scrolls the viewport.
func (m Model) detailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.detailOpen = false
		return m, nil
	case key.Matches(msg, m.keys.Edit):
		return m, editCmd(m.editor, m.detailRow.Path)
	}
	var cmd tea.Cmd
	m.detailVP, cmd = m.detailVP.Update(msg)
	return m, cmd
}

// item is one selectable line of the board view: a sub-board or a task row.
type item struct {
	sub *Sub
	row *Row
}

// items lists the board's selectable lines: sub-boards first, then every
// section's rows in board order.
func (m Model) items() []item {
	b, ok := m.board()
	if !ok {
		return nil
	}
	var its []item
	for i := range b.Subs {
		its = append(its, item{sub: &b.Subs[i]})
	}
	for si := range b.Sections {
		for ri := range b.Sections[si].Rows {
			its = append(its, item{row: &b.Sections[si].Rows[ri]})
		}
	}
	return its
}

// board returns the view model of the board on screen.
func (m Model) board() (Board, bool) {
	b, ok := m.snap.Boards[m.path]
	return b, ok
}

// selected returns the item under the cursor.
func (m Model) selected() (item, bool) {
	its := m.items()
	c := m.cursor()
	if c >= len(its) {
		return item{}, false
	}
	return its[c], true
}

// cursor returns the current board's cursor, clamped to its items.
func (m Model) cursor() int {
	return clamp(m.cursors[m.path], 0, len(m.items())-1)
}

// moveCursor shifts the current board's cursor by delta, clamped.
func (m Model) moveCursor(delta int) Model {
	m.cursors[m.path] = clamp(m.cursor()+delta, 0, len(m.items())-1)
	return m
}

// open descends into the selected sub-board or opens the selected task's
// detail; rows whose file is missing have nothing to open.
func (m Model) open() Model {
	it, ok := m.selected()
	if !ok {
		return m
	}
	if it.sub != nil {
		m.path = it.sub.Path
		return m
	}
	if it.row.Path == "" {
		return m
	}
	return m.openDetail(*it.row, 0)
}

// back climbs to the parent board; a no-op on the root.
func (m Model) back() Model {
	if b, ok := m.board(); ok && b.Parent != "" {
		m.path = b.Parent
	}
	return m
}

// openDetail (re)builds the detail view for row at the current size,
// restoring the scroll offset.
func (m Model) openDetail(row Row, offset int) Model {
	w, h := m.dims()
	vp := viewport.New(w, max(h-detailChrome, 1))
	vp.SetContent(renderMarkdown(row.Content, w, m.theme))
	vp.SetYOffset(offset)
	m.detailOpen, m.detailRow, m.detailVP = true, row, vp
	return m
}

// withSnap applies a re-scan: on error the last good snapshot stays on
// screen with the error in the footer; on success the view — including an
// open detail — re-renders from the new truth.
func (m Model) withSnap(msg snapMsg) Model {
	if msg.err != nil {
		m.scanErr = msg.err.Error()
		return m
	}
	m.scanErr = ""
	m.snap = msg.snap
	for m.path != "." {
		if _, ok := m.snap.Boards[m.path]; ok {
			break
		}
		m.path = textualParent(m.path)
	}
	if m.detailOpen {
		row, ok := m.findRow(m.detailRow.ID)
		if !ok {
			m.detailOpen = false
			return m
		}
		m = m.openDetail(row, m.detailVP.YOffset)
	}
	return m
}

// findRow locates a task by id anywhere in the snapshot.
func (m Model) findRow(id string) (Row, bool) {
	for _, b := range m.snap.Boards {
		for _, s := range b.Sections {
			for _, r := range s.Rows {
				if r.ID == id && r.Path != "" {
					return r, true
				}
			}
		}
	}
	return Row{}, false
}

// dims returns the terminal size, defaulting to 80×24 before the first
// tea.WindowSizeMsg.
func (m Model) dims() (w, h int) {
	if m.width <= 0 || m.height <= 0 {
		return 80, 24
	}
	return m.width, m.height
}

// waitRefresh blocks on the watcher channel and turns one notification into
// a refreshMsg; a closed channel ends the wait silently.
func waitRefresh(c <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if _, ok := <-c; !ok {
			return nil
		}
		return refreshMsg{}
	}
}

// scanCmd re-reads the whole tree off the update loop.
func scanCmd(root string) tea.Cmd {
	return func() tea.Msg {
		snap, err := Scan(root)
		return snapMsg{snap: snap, err: err}
	}
}

// editCmd suspends the program and opens path in the configured editor,
// which may carry arguments ("code -w").
func editCmd(editor, path string) tea.Cmd {
	if path == "" {
		return nil
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	c := exec.Command(parts[0], append(parts[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return editedMsg{err: err} })
}

// textualParent trims the last path segment: "a/b" → "a", "a" → ".".
func textualParent(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "."
	}
	return p[:i]
}

// clamp bounds v to [lo, hi]; hi below lo collapses to lo.
func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
