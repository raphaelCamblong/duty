package tui

import (
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
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

// focusArea names the panel holding keyboard focus.
type focusArea int

const (
	focusList focusArea = iota
	focusPreview
)

// Preview subject kinds: the open preview shows either a task's rendered
// markdown or a track's summary card.
const (
	previewTask  = "task"
	previewTrack = "track"
)

// Model is the Bubble Tea model of the viewer: a full-width entry list under
// a subtree-state header, opening a split preview on demand (a task rendered,
// a track summarized). Update is a pure transition — filesystem reads happen
// in commands, writes nowhere.
type Model struct {
	fsys    fsys.FS
	root    string
	editor  string
	theme   string
	keys    keyMap
	help    help.Model
	zones   *zone.Manager
	refresh <-chan struct{}

	snap    Snapshot
	scanErr string
	path    string
	memory  map[string]int
	width   int
	height  int

	list             list.Model
	focus            focusArea
	preview          viewport.Model
	previewOpen      bool
	previewKind      string
	previewArg       string
	previewTitleText string
	renderer         *glamour.TermRenderer
	rendererWidth    int

	spring       harmonica.Spring
	scroll       float64
	scrollVel    float64
	scrollTarget int
	animating    bool
	lastClick    int
	lastClickAt  time.Time
}

// New scans the tree under root and returns a model opened on the root
// track, styled per cfg.
func New(f fsys.FS, root string, cfg config.Config) (Model, error) {
	snap, err := Scan(f, root)
	if err != nil {
		return Model{}, err
	}
	m := Model{
		fsys:      f,
		root:      root,
		editor:    cfg.Editor,
		theme:     cfg.TUI.Theme,
		keys:      defaultKeys(),
		help:      help.New(),
		zones:     zone.New(),
		snap:      snap,
		path:      ".",
		memory:    map[string]int{},
		list:      newList(),
		spring:    harmonica.NewSpring(harmonica.FPS(scrollFPS), scrollFreq, 1.0),
		lastClick: -1,
	}
	m, _ = m.rebuildList()
	return m.fixSelection().layout(), nil
}

// newList returns the bare bubbles list the left panel wraps: no chrome of
// its own (the model owns header, footer, and quitting), fuzzy filter on.
func newList() list.Model {
	l := list.New(nil, compactDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("match", "matches")
	l.DisableQuitKeybindings()
	l.FilterInput.Prompt = "/ "
	l.FilterInput.PromptStyle = accentStyle
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.Styles.NoItems = dimStyle.Padding(1, 2)
	return l
}

// Close releases the model's zone manager; call it once after the program
// exits.
func (m Model) Close() {
	if m.zones != nil {
		m.zones.Close()
	}
}

// BoardPath returns the path of the track on screen ("." = root).
func (m Model) BoardPath() string { return m.path }

// ScrollTarget returns the preview's spring target top line (for tests).
func (m Model) ScrollTarget() int { return m.scrollTarget }

// HelpExpanded reports whether the full "?" key grid is showing.
func (m Model) HelpExpanded() bool { return m.help.ShowAll }

// PreviewFocused reports whether the preview panel holds focus.
func (m Model) PreviewFocused() bool { return m.focus == focusPreview }

// PreviewOpen reports whether the split preview is showing (false while
// browsing the full-width list).
func (m Model) PreviewOpen() bool { return m.previewOpen }

// Cursor returns the selected item index — sub-tracks then tasks, section
// headers not counted.
func (m Model) Cursor() int {
	items := m.list.VisibleItems()
	idx := m.list.Index()
	c := 0
	for i := 0; i < idx && i < len(items); i++ {
		if e, ok := items[i].(entry); ok && e.selectable() {
			c++
		}
	}
	return c
}

// SelectedID returns the selection's task id or track path, "" for none.
func (m Model) SelectedID() string {
	e, ok := m.selectedEntry()
	switch {
	case !ok:
		return ""
	case e.track != nil:
		return e.track.Path
	default:
		return e.task.ID
	}
}

// DetailID returns the id of the task the open preview shows, "" if the
// preview is closed or showing a track card.
func (m Model) DetailID() string {
	if !m.previewOpen {
		return ""
	}
	if m.previewKind == previewTask {
		return m.previewArg
	}
	return ""
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
		m.help.Width = msg.Width
		return m.layout(), nil
	case refreshMsg:
		if m.refresh == nil {
			return m, scanCmd(m.fsys, m.root)
		}
		return m, tea.Batch(scanCmd(m.fsys, m.root), waitRefresh(m.refresh))
	case snapMsg:
		return m.withSnap(msg)
	case editedMsg:
		return m, scanCmd(m.fsys, m.root)
	case scrollTickMsg:
		return m.stepScroll()
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m.updateList(msg)
}

// handleKey routes a key press: everything to the filter input while one is
// being typed, global keys next, panel actions after, then the focused panel.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.list.SettingFilter() {
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m.updateList(msg)
	}
	if model, cmd, ok := m.handleGlobalKey(msg); ok {
		return model, cmd
	}
	if model, cmd, ok := m.handleActionKey(msg); ok {
		return model, cmd
	}
	if m.focus == focusPreview {
		return m.scrollPreview(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Down):
		return m.moveSelection(1), nil
	case key.Matches(msg, m.keys.Up):
		return m.moveSelection(-1), nil
	}
	return m.updateList(msg)
}

// handleGlobalKey handles the keys that apply regardless of the focused
// panel: quit, help, refresh, focus toggle, filter.
func (m Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m.layout(), nil, true
	case key.Matches(msg, m.keys.Refresh):
		return m, scanCmd(m.fsys, m.root), true
	case key.Matches(msg, m.keys.Focus):
		if m.previewOpen {
			m.focus = otherFocus(m.focus)
		}
		return m, nil, true
	case key.Matches(msg, m.keys.Filter):
		model, cmd := m.filterList(msg)
		return model, cmd, true
	}
	return m, nil, false
}

// handleActionKey handles the keys acting on the selected entry: edit, back,
// open.
func (m Model) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Edit):
		if e, ok := m.selectedEntry(); ok && e.task != nil {
			return m, editCmd(m.editor, e.task.Path), true
		}
		return m, nil, true
	case key.Matches(msg, m.keys.Back):
		model, cmd := m.back()
		return model, cmd, true
	case key.Matches(msg, m.keys.Open):
		if m.focus == focusList {
			return m.open(), nil, true
		}
		return m, nil, true
	}
	return m, nil, false
}

// filterList moves focus to the list and starts the fuzzy filter.
func (m Model) filterList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.focus = focusList
	return m.updateList(msg)
}

// scrollPreview forwards a key to the preview viewport and settles the
// scroll position.
func (m Model) scrollPreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m.settleAt(m.preview.YOffset), cmd
}

// otherFocus toggles between the two panels.
func otherFocus(f focusArea) focusArea {
	if f == focusList {
		return focusPreview
	}
	return focusList
}

// updateList forwards a message to the bubbles list, then re-aims the
// selection. Browsing never renders the preview (§8).
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m.fixSelection(), cmd
}

// selectedEntry returns the selected track or task entry; headers and an
// empty list yield none.
func (m Model) selectedEntry() (entry, bool) {
	e, ok := m.list.SelectedItem().(entry)
	if !ok || !e.selectable() {
		return entry{}, false
	}
	return e, true
}

// board returns the view model of the track on screen.
func (m Model) board() (Board, bool) {
	b, ok := m.snap.Boards[m.path]
	return b, ok
}

// moveSelection shifts the selection by delta, skipping section headers and
// stopping at the list's edges.
func (m Model) moveSelection(delta int) Model {
	items := m.list.VisibleItems()
	for i := m.list.Index() + delta; i >= 0 && i < len(items); i += delta {
		if e, ok := items[i].(entry); ok && e.selectable() {
			m.list.Select(i)
			break
		}
	}
	return m
}

// fixSelection nudges a selection off a section header onto the nearest
// selectable entry below it, or above when the header is last, first pulling
// an out-of-range cursor back onto the visible list (filtering can strand it).
func (m Model) fixSelection() Model {
	items := m.list.VisibleItems()
	if len(items) == 0 {
		return m
	}
	idx := m.list.Index()
	if idx < 0 || idx >= len(items) {
		idx = clamp(idx, 0, len(items)-1)
		m.list.Select(idx)
	}
	if e, ok := items[idx].(entry); ok && e.selectable() {
		return m
	}
	for _, delta := range []int{1, -1} {
		for i := idx + delta; i >= 0 && i < len(items); i += delta {
			if e, ok := items[i].(entry); ok && e.selectable() {
				m.list.Select(i)
				return m
			}
		}
	}
	return m
}

// open acts on the selection: a task always opens the split preview; a track
// opens its summary card only when the preview is already open, otherwise it
// descends into the track (§8).
func (m Model) open() Model {
	e, ok := m.selectedEntry()
	if !ok {
		return m
	}
	if e.track != nil {
		if m.previewOpen {
			return m.showPreview(previewTrack, e.track.Path)
		}
		return m.enterBoard(e.track.Path)
	}
	return m.showPreview(previewTask, e.task.ID)
}

// showPreview opens the split on the subject (kind previewTask or previewTrack,
// arg the id or track path): it sizes the two panels, renders the subject, and
// focuses the preview.
func (m Model) showPreview(kind, arg string) Model {
	m.previewOpen = true
	m.focus = focusPreview
	m.previewKind = kind
	m.previewArg = arg
	m = m.layout()
	return m.renderPreview(true)
}

// renderPreview fills the viewport from the open subject, resolved against the
// current snapshot; reset returns to the top, otherwise the scroll offset
// survives (a re-scan of the same subject). A no-op while browsing.
func (m Model) renderPreview(reset bool) Model {
	if !m.previewOpen {
		return m
	}
	off := m.preview.YOffset
	if reset {
		off = 0
	}
	m, body := m.previewContent()
	m.previewTitleText = m.previewTitle()
	m.preview.SetContent(body)
	m.preview.SetYOffset(off)
	return m.settleAt(m.preview.YOffset)
}

// findRowBoard resolves a task id to its row and the board it sits in, used
// by the preview header for the row's track title.
func (m Model) findRowBoard(id string) (Row, Board, bool) {
	for _, b := range m.snap.Boards {
		for _, s := range b.Sections {
			for _, r := range s.Rows {
				if r.ID == id {
					return r, b, true
				}
			}
		}
	}
	return Row{}, Board{}, false
}

// findSub resolves a track path to its sub-track view anywhere in the snapshot.
func (m Model) findSub(path string) (Sub, bool) {
	for _, b := range m.snap.Boards {
		for _, s := range b.Subs {
			if s.Path == path {
				return s, true
			}
		}
	}
	return Sub{}, false
}

// back closes the open preview (to full-width browsing), then clears an
// applied filter, then climbs to the parent track; a no-op on an unfiltered
// root list.
func (m Model) back() (tea.Model, tea.Cmd) {
	if m.previewOpen {
		m.previewOpen = false
		m.focus = focusList
		return m.layout(), nil
	}
	if m.list.IsFiltered() {
		m.list.ResetFilter()
		return m.fixSelection(), nil
	}
	if b, ok := m.board(); ok && b.Parent != "" {
		return m.enterBoard(b.Parent), nil
	}
	return m, nil
}

// enterBoard switches the view to the track at path, remembering the old
// selection and restoring any previous one on the target. Descending returns
// to full-width browsing.
func (m Model) enterBoard(path string) Model {
	m.memory[m.path] = m.Cursor()
	m.path = path
	m.list.ResetFilter()
	m, _ = m.rebuildList()
	m = m.selectNth(m.memory[path]).fixSelection()
	m.previewOpen = false
	m.focus = focusList
	return m.layout()
}

// selectNth selects the nth selectable entry (headers not counted).
func (m Model) selectNth(n int) Model {
	c := 0
	for i, it := range m.list.VisibleItems() {
		e, ok := it.(entry)
		if !ok || !e.selectable() {
			continue
		}
		if c == n {
			m.list.Select(i)
			return m
		}
		c++
	}
	return m
}

// rebuildList reloads the left panel's entries and column widths from the
// snapshot; the returned command re-runs an active filter.
func (m Model) rebuildList() (Model, tea.Cmd) {
	b, ok := m.board()
	m.list.SetDelegate(newDelegate(m.zones, b))
	if !ok {
		return m, m.list.SetItems(nil)
	}
	return m, m.list.SetItems(boardEntries(b))
}

// withSnap applies a re-scan: on error the last good snapshot stays on
// screen with the error in the footer; on success every panel re-renders
// from the new truth, keeping selection, filter, and preview scroll.
func (m Model) withSnap(msg snapMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.scanErr = msg.err.Error()
		return m, nil
	}
	m.scanErr = ""
	m.snap = msg.snap
	for m.path != "." {
		if _, ok := m.snap.Boards[m.path]; ok {
			break
		}
		m.path = textualParent(m.path)
	}
	m, cmd := m.rebuildList()
	return m.fixSelection().renderPreview(false), cmd
}

// dims returns the terminal size, defaulting to 80×24 before the first
// tea.WindowSizeMsg.
func (m Model) dims() (w, h int) {
	if m.width <= 0 || m.height <= 0 {
		return 80, 24
	}
	return m.width, m.height
}

// wide reports whether the terminal fits the two-panel layout.
func (m Model) wide() bool {
	w, _ := m.dims()
	return w >= twoPanelMinWidth
}

// split reports whether the side-by-side layout shows: an open preview on a
// wide-enough terminal.
func (m Model) split() bool {
	return m.previewOpen && m.wide()
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
func scanCmd(f fsys.FS, root string) tea.Cmd {
	return func() tea.Msg {
		snap, err := Scan(f, root)
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
