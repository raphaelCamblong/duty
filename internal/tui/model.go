package tui

import (
	"os/exec"
	"path"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/task"
)

// refreshMsg reports a debounced filesystem change from the watcher.
type refreshMsg struct{}

type snapMsg struct {
	snap Snapshot
	err  error
}

type editedMsg struct{ err error }

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
	theme   Theme
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
	spinner          spinner.Model
	spinning         bool
	showAge          bool
	showGates        bool
	showArchive      bool
	statusSort       bool
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

// New scans the tree under root and returns a model opened on the root track,
// styled per cfg: the color palette overlaid from [tui.palette] (a malformed
// color errors) and the dark/light mode from [tui].theme.
func New(filesystem fsys.FS, root string, cfg config.Config) (Model, error) {
	theme, err := themeFromConfig(cfg.TUI.Palette, cfg.TUI.Theme != "light")
	if err != nil {
		return Model{}, err
	}
	snap, err := Scan(filesystem, root, false)
	if err != nil {
		return Model{}, err
	}
	model := Model{
		fsys:       filesystem,
		root:       root,
		editor:     cfg.Editor,
		theme:      theme,
		keys:       defaultKeys(),
		help:       help.New(),
		zones:      zone.New(),
		snap:       snap,
		path:       ".",
		memory:     map[string]int{},
		list:       newList(theme),
		spinner:    newSpinner(theme),
		showAge:    true,
		showGates:  true,
		statusSort: true,
		spring:     harmonica.NewSpring(harmonica.FPS(scrollFPS), scrollFreq, 1.0),
		lastClick:  -1,
	}
	model, _ = model.rebuildList()
	return model.fixSelection().layout(), nil
}

func newList(theme Theme) list.Model {
	listModel := list.New(nil, compactDelegate{theme: theme}, 0, 0)
	listModel.SetShowTitle(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowHelp(false)
	listModel.SetFilteringEnabled(true)
	listModel.SetStatusBarItemName("match", "matches")
	listModel.DisableQuitKeybindings()
	listModel.FilterInput.Prompt = "/ "
	fs := listModel.FilterInput.Styles()
	fs.Focused.Prompt = theme.accent()
	fs.Blurred.Prompt = theme.accent()
	listModel.FilterInput.SetStyles(fs)
	listModel.Styles.TitleBar = lipgloss.NewStyle()
	listModel.Styles.NoItems = theme.dim().Padding(1, 2)
	return listModel
}

// newSpinner returns the one shared in-progress spinner: a one-cell MiniDot
// tinted with the theme's in-progress ink (peach on dark, blue on light).
func newSpinner(theme Theme) spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.statusStyle(task.StatusInProgress)),
	)
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

func (m Model) HelpExpanded() bool { return m.help.ShowAll }

func (m Model) PreviewFocused() bool { return m.focus == focusPreview }

func (m Model) PreviewOpen() bool { return m.previewOpen }

func (m Model) ShowAge() bool { return m.showAge }

// ShowArchive reports whether the session-only archive view is on.
func (m Model) ShowArchive() bool { return m.showArchive }

// Cursor returns the selected item index — sub-tracks then tasks, section
// headers not counted.
func (m Model) Cursor() int {
	items := m.list.VisibleItems()
	idx := m.list.Index()
	count := 0
	for i := 0; i < idx && i < len(items); i++ {
		if item, ok := items[i].(entry); ok && item.selectable() {
			count++
		}
	}
	return count
}

func (m Model) SelectedID() string {
	item, ok := m.selectedEntry()
	switch {
	case !ok:
		return ""
	case item.track != nil:
		return item.track.Path
	default:
		return item.task.ID
	}
}

func (m Model) DetailID() string {
	if !m.previewOpen {
		return ""
	}
	if m.previewKind == previewTask {
		return m.previewArg
	}
	return ""
}

func (m Model) Init() tea.Cmd {
	if m.refresh == nil {
		return nil
	}
	return waitRefresh(m.refresh)
}

// Spinning reports whether the in-progress spinner's tick loop is live (for
// tests): a quiet snapshot schedules no ticks at all.
func (m Model) Spinning() bool { return m.spinning }

// arm starts the spinner's tick loop when the snapshot holds an in-progress
// task and no loop is already running, batching the first tick with base; an
// already-running loop or a quiet board returns base untouched. It never stops
// a loop — a live loop halts itself in onSpinnerTick once the last in-progress
// task leaves the snapshot.
func (m Model) arm(base tea.Cmd) (tea.Model, tea.Cmd) {
	if m.spinning || !m.snap.anyInProgress() {
		return m, base
	}
	m.spinning = true
	return m, tea.Batch(base, m.spinner.Tick)
}

// onSpinnerTick advances the shared spinner one frame while in-progress work
// remains, rescheduling the next tick; once the last in-progress task leaves the
// snapshot it drops the tick — scheduling nothing further — so a quiet board
// costs zero re-renders.
func (m Model) onSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if !m.snap.anyInProgress() {
		m.spinning = false
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m.reskinList(), cmd
}

func (m Model) spinnerGlyph() string {
	if !m.snap.anyInProgress() {
		return ""
	}
	return m.spinner.View()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(msg.Width)
		return m.resizeGates(msg.Width).layout().arm(nil)
	case spinner.TickMsg:
		return m.onSpinnerTick(msg)
	case refreshMsg:
		if m.refresh == nil {
			return m, scanCmd(m.fsys, m.root, m.showArchive)
		}
		return m, tea.Batch(scanCmd(m.fsys, m.root, m.showArchive), waitRefresh(m.refresh))
	case snapMsg:
		return m.withSnap(msg)
	case editedMsg:
		return m, scanCmd(m.fsys, m.root, m.showArchive)
	case scrollTickMsg:
		return m.stepScroll()
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m.updateList(msg)
}

// handleKey routes a key press: everything to the filter input while one is
// being typed, global keys next, panel actions after, then the focused panel.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.list.SettingFilter() {
		if msg.String() == "ctrl+c" {
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

func (m Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m.layout(), nil, true
	case key.Matches(msg, m.keys.Refresh):
		return m, scanCmd(m.fsys, m.root, m.showArchive), true
	case key.Matches(msg, m.keys.Archive):
		return m.toggleArchive()
	case key.Matches(msg, m.keys.Focus):
		if m.previewOpen {
			m.focus = otherFocus(m.focus)
		}
		return m, nil, true
	case key.Matches(msg, m.keys.Filter):
		if !anySelectable(m.list.Items()) {
			return m, nil, true
		}
		model, cmd := m.filterList(msg)
		return model, cmd, true
	case key.Matches(msg, m.keys.Age):
		m.showAge = !m.showAge
		return m.reskinList().layout(), nil, true
	case key.Matches(msg, m.keys.Sort):
		m.statusSort = !m.statusSort
		model, cmd := m.rebuildList()
		return model.fixSelection().layout(), cmd, true
	}
	return m, nil, false
}

// toggleArchive flips the session-only archive view. The list rebuilds at once
// — the hiding rule reads archived counts already in the snapshot — while a
// re-scan loads the archived rows (toggling on) or drops them (toggling off),
// so archive contents are read only while the toggle is on.
func (m Model) toggleArchive() (tea.Model, tea.Cmd, bool) {
	m.showArchive = !m.showArchive
	model, cmd := m.rebuildList()
	scan := scanCmd(m.fsys, m.root, m.showArchive)
	return model.fixSelection().layout(), tea.Batch(cmd, scan), true
}

func (m Model) resizeGates(width int) Model {
	if show := width >= narrowCols; show != m.showGates {
		m.showGates = show
		return m.reskinList()
	}
	return m
}

// reskinList swaps in a fresh list delegate reflecting the current age and gate
// visibility, leaving items, selection, and any filter untouched.
func (m Model) reskinList() Model {
	board, _ := m.board()
	m.list.SetDelegate(newDelegate(m.theme, m.zones, board, viewOpts{showAge: m.showAge, showGates: m.showGates, showArchive: m.showArchive, now: time.Now(), glyph: m.spinnerGlyph()}))
	return m
}

func (m Model) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Edit):
		if item, ok := m.selectedEntry(); ok && item.task != nil && !item.archived {
			return m, editCmd(m.editor, item.task.Path), true
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

func (m Model) filterList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.focus = focusList
	return m.updateList(msg)
}

func (m Model) scrollPreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m.settleAt(m.preview.YOffset()), cmd
}

func otherFocus(focus focusArea) focusArea {
	if focus == focusList {
		return focusPreview
	}
	return focusList
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m.fixSelection(), cmd
}

func (m Model) selectedEntry() (entry, bool) {
	item, ok := m.list.SelectedItem().(entry)
	if !ok || !item.selectable() {
		return entry{}, false
	}
	return item, true
}

func (m Model) board() (Board, bool) {
	board, ok := m.snap.Boards[m.path]
	return board, ok
}

func (m Model) moveSelection(delta int) Model {
	items := m.list.VisibleItems()
	for i := m.list.Index() + delta; i >= 0 && i < len(items); i += delta {
		if item, ok := items[i].(entry); ok && item.selectable() {
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
	if item, ok := items[idx].(entry); ok && item.selectable() {
		return m
	}
	for _, delta := range []int{1, -1} {
		if i, ok := selectableFrom(items, idx+delta, delta); ok {
			m.list.Select(i)
			return m
		}
	}
	return m
}

// selectableFrom returns the index of the first selectable entry from start,
// stepping by delta, or false when the scan runs off either end.
func selectableFrom(items []list.Item, start, delta int) (int, bool) {
	for i := start; i >= 0 && i < len(items); i += delta {
		if item, ok := items[i].(entry); ok && item.selectable() {
			return i, true
		}
	}
	return 0, false
}

// open acts on the selection: a task always opens the split preview; a track
// opens its summary card only when the preview is already open, otherwise it
// descends into the track (§8).
func (m Model) open() Model {
	item, ok := m.selectedEntry()
	if !ok {
		return m
	}
	if item.track != nil {
		if m.previewOpen {
			return m.showPreview(previewTrack, item.track.Path)
		}
		return m.enterBoard(item.track.Path)
	}
	return m.showPreview(previewTask, item.task.ID)
}

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
	off := m.preview.YOffset()
	if reset {
		off = 0
	}
	m, body := m.previewContent()
	m.previewTitleText = m.previewTitle()
	m.preview.SetContent(body)
	m.preview.SetYOffset(off)
	return m.settleAt(m.preview.YOffset())
}

// findRowBoard resolves a task id to its row and the board it sits in, used
// by the preview header for the row's track title. Archived rows are searched
// too, so an archived task opens the same read-only preview.
func (m Model) findRowBoard(id string) (Row, Board, bool) {
	for _, board := range m.snap.Boards {
		if row, ok := boardRow(board, id); ok {
			return row, board, true
		}
	}
	return Row{}, Board{}, false
}

// boardRow finds a row by id anywhere in a board: any section first, then the
// archived rows.
func boardRow(board Board, id string) (Row, bool) {
	for _, section := range board.Sections {
		if row, ok := rowByID(section.Rows, id); ok {
			return row, true
		}
	}
	return rowByID(board.Archived, id)
}

func rowByID(rows []Row, id string) (Row, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return Row{}, false
}

func (m Model) findSub(path string) (Sub, bool) {
	for _, board := range m.snap.Boards {
		if sub, ok := subByPath(board.Subs, path); ok {
			return sub, true
		}
	}
	return Sub{}, false
}

func subByPath(subs []Sub, path string) (Sub, bool) {
	for _, sub := range subs {
		if sub.Path == path {
			return sub, true
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
	if board, ok := m.board(); ok && board.Parent != "" {
		return m.enterBoard(board.Parent), nil
	}
	return m, nil
}

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

func (m Model) selectNth(target int) Model {
	count := 0
	for i, it := range m.list.VisibleItems() {
		item, ok := it.(entry)
		if !ok || !item.selectable() {
			continue
		}
		if count == target {
			m.list.Select(i)
			return m
		}
		count++
	}
	return m
}

// rebuildList reloads the left panel's entries and column widths from the
// snapshot; the returned command re-runs an active filter.
func (m Model) rebuildList() (Model, tea.Cmd) {
	m = m.reskinList()
	board, ok := m.board()
	if !ok {
		return m, m.list.SetItems(nil)
	}
	return m, m.list.SetItems(boardEntries(board, m.statusSort, m.showArchive))
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
		m.path = path.Dir(m.path)
	}
	m, cmd := m.rebuildList()
	return m.fixSelection().renderPreview(false).arm(cmd)
}

func (m Model) dims() (width, height int) {
	if m.width <= 0 || m.height <= 0 {
		return 80, 24
	}
	return m.width, m.height
}

func (m Model) wide() bool {
	width, _ := m.dims()
	return width >= twoPanelMinWidth
}

func (m Model) split() bool {
	return m.previewOpen && m.wide()
}

// waitRefresh blocks on the watcher channel and turns one notification into
// a refreshMsg; a closed channel ends the wait silently.
func waitRefresh(refresh <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if _, ok := <-refresh; !ok {
			return nil
		}
		return refreshMsg{}
	}
}

// scanCmd re-reads the whole tree off the update loop, including archived rows
// only when includeArchive is set.
func scanCmd(filesystem fsys.FS, root string, includeArchive bool) tea.Cmd {
	return func() tea.Msg {
		snap, err := Scan(filesystem, root, includeArchive)
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
	command := exec.Command(parts[0], append(parts[1:], path)...)
	return tea.ExecProcess(command, func(err error) tea.Msg { return editedMsg{err: err} })
}

// clamp bounds v to [lo, hi]; hi below lo collapses to lo.
func clamp(value, lo, hi int) int {
	return max(lo, min(value, hi))
}
