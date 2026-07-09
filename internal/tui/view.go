package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/raphaelCamblong/duty/internal/task"
)

const (
	statusColWidth = len(task.StatusInProgress)
	gatesColWidth  = 5
	minTitleWidth  = 8
	// twoPanelMinWidth is the narrowest terminal that fits the master-detail
	// layout; below it the view falls back to a single panel.
	twoPanelMinWidth = 80
	// minLeftWidth floors the left panel so entries stay readable.
	minLeftWidth = 30
	// minBarWidth is the shortest distribution bar worth drawing.
	minBarWidth = 8
)

// rollupOrder is the status sequence for track rollups and summaries: active
// work first, then queued, blocked, and finished — matching the header bar's
// colors.
var rollupOrder = []string{task.StatusInProgress, task.StatusTodo, task.StatusBlocked, task.StatusDone}

// zoneList and zonePreview are the BubbleZone names of the two panels.
const (
	zoneList    = "panel-list"
	zonePreview = "panel-preview"
)

var (
	colAccent = lipgloss.AdaptiveColor{Light: "62", Dark: "111"}
	colDim    = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	colYellow = lipgloss.AdaptiveColor{Light: "136", Dark: "220"}
	colRed    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	colGreen  = lipgloss.AdaptiveColor{Light: "28", Dark: "78"}

	headerBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
	focusedBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
	blurredBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colDim).Padding(0, 1)
	crumbStyle   = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	accentStyle  = lipgloss.NewStyle().Foreground(colAccent)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(colDim)
	dimStyle     = lipgloss.NewStyle().Foreground(colDim)
	errStyle     = lipgloss.NewStyle().Foreground(colRed)
	selStyle     = lipgloss.NewStyle().Bold(true)
	driftStyle   = lipgloss.NewStyle().Foreground(colRed)
	yellowStyle  = lipgloss.NewStyle().Foreground(colYellow)
	redStyle     = lipgloss.NewStyle().Foreground(colRed)
	greenStyle   = lipgloss.NewStyle().Foreground(colGreen)
)

// statusStyle maps a task status to its color: todo dim, in-progress
// yellow, blocked red, done green.
func statusStyle(status string) lipgloss.Style {
	switch status {
	case task.StatusInProgress:
		return yellowStyle
	case task.StatusBlocked:
		return redStyle
	case task.StatusDone:
		return greenStyle
	}
	return dimStyle
}

// statusColor is the fill for a status's segment of a distribution bar,
// matching statusStyle's foregrounds.
func statusColor(status string) lipgloss.TerminalColor {
	switch status {
	case task.StatusInProgress:
		return colYellow
	case task.StatusBlocked:
		return colRed
	case task.StatusDone:
		return colGreen
	}
	return colDim
}

// panelBox is a panel's border style: accent when focused, dim otherwise.
func panelBox(focused bool) lipgloss.Style {
	if focused {
		return focusedBox
	}
	return blurredBox
}

// View renders the current frame: header, the two panels (or the single
// narrow-terminal panel), and the help footer. The zone manager's Scan
// registers the hit-zones and strips their markers.
func (m Model) View() string {
	w, _ := m.dims()
	var body string
	switch {
	case m.wide():
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.leftPanel(), m.rightPanel())
	case m.focus == focusPreview:
		title := ansi.Truncate(" "+m.previewTitle(), max(w-1, 1), "…")
		body = lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	default:
		body = m.list.View()
	}
	frame := lipgloss.JoinVertical(lipgloss.Left, m.headerView(w), body, m.footerView(w))
	return m.zones.Scan(frame)
}

// layout sizes the list and preview to the current terminal and chrome
// heights, then re-renders the preview at the new width.
func (m Model) layout() Model {
	w, h := m.dims()
	bodyH := max(h-lipgloss.Height(m.headerView(w))-lipgloss.Height(m.footerView(w)), 3)
	if m.wide() {
		lw := leftWidth(w)
		m.list.SetSize(lw-4, bodyH-2)
		m.preview.Width = max(w-lw-4, 1)
		m.preview.Height = max(bodyH-3, 1)
	} else {
		m.list.SetSize(w, bodyH)
		m.preview.Width = max(w-2, 1)
		m.preview.Height = max(bodyH-1, 1)
	}
	return m.syncPreview(true)
}

// leftWidth is the left panel's total width: ~38% of the terminal, floored.
func leftWidth(w int) int {
	return max(w*38/100, minLeftWidth)
}

// leftPanel is the entry list in its focus-colored border, a full-panel
// mouse zone.
func (m Model) leftPanel() string {
	box := panelBox(m.focus == focusList).Width(m.list.Width() + 2).Height(m.list.Height())
	return m.zones.Mark(zoneList, box.Render(m.list.View()))
}

// rightPanel is the preview — pinned title line over the viewport — in its
// focus-colored border, a full-panel mouse zone.
func (m Model) rightPanel() string {
	title := ansi.Truncate(m.previewTitle(), m.preview.Width, "…")
	box := panelBox(m.focus == focusPreview).Width(m.preview.Width + 2).Height(m.preview.Height + 1)
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	return m.zones.Mark(zonePreview, box.Render(content))
}

// headerView is the rounded box holding the breadcrumb and the current
// track's subtree state: per-status counts plus the distribution bar (§8).
func (m Model) headerView(w int) string {
	inner := max(w-4, 1)
	b, _ := m.board()
	content := lipgloss.JoinVertical(lipgloss.Left,
		ansi.Truncate(crumbStyle.Render(m.breadcrumb()), inner, "…"),
		stateLine(b, inner),
	)
	return headerBox.Width(max(w-2, 1)).Render(content)
}

// stateLine renders a board's subtree per-status counts in status colors,
// with the ntcharts distribution bar filling the rest of the line.
func stateLine(b Board, w int) string {
	rollup := rollupOrEmpty(b.Counts)
	barW := w - lipgloss.Width(rollup) - 2
	if barW < minBarWidth {
		return ansi.Truncate(rollup, w, "…")
	}
	return rollup + "  " + statusBar(b.Counts, barW)
}

// statusBar renders per-status counts as one horizontal ntcharts bar w cells
// wide; no tasks shows a faint rule.
func statusBar(counts map[string]int, w int) string {
	total := 0
	for _, st := range rollupOrder {
		total += counts[st]
	}
	if total == 0 {
		return dimStyle.Render(strings.Repeat("╌", w))
	}
	bar := barchart.New(w, 1,
		barchart.WithHorizontalBars(),
		barchart.WithNoAxis(),
		barchart.WithBarWidth(1),
		barchart.WithDataSet([]barchart.BarData{barData(counts)}),
	)
	bar.Draw()
	return bar.View()
}

// barData turns status counts into one stacked horizontal bar, segments in
// lifecycle order and colored to match the row statuses.
func barData(counts map[string]int) barchart.BarData {
	seg := func(status string) barchart.BarValue {
		c := statusColor(status)
		return barchart.BarValue{
			Name:  status,
			Value: float64(counts[status]),
			Style: lipgloss.NewStyle().Foreground(c).Background(c),
		}
	}
	return barchart.BarData{Values: []barchart.BarValue{
		seg(task.StatusTodo), seg(task.StatusInProgress),
		seg(task.StatusBlocked), seg(task.StatusDone),
	}}
}

// footerView is the bubbles/help hint bar, topped by the last scan error
// when one is pending.
func (m Model) footerView(w int) string {
	if m.scanErr == "" {
		return m.helpView(w)
	}
	err := " " + errStyle.Render(ansi.Truncate(m.scanErr, max(w-2, 1), "…"))
	return lipgloss.JoinVertical(lipgloss.Left, err, m.helpView(w))
}

// helpView renders the bubbles/help hint bar (short, or the full grid after
// "?"), sized to the terminal.
func (m Model) helpView(w int) string {
	h := m.help
	h.Width = w
	return " " + h.View(m.keys)
}

// previewTitle is the pinned line above the preview: id, status, gates, and
// drift for a task; name and title for a track.
func (m Model) previewTitle() string {
	e, ok := m.selectedEntry()
	switch {
	case !ok:
		return dimStyle.Render("nothing selected")
	case e.track != nil:
		return accentStyle.Render(e.track.Name) + "  " + selStyle.Render(e.track.Title)
	default:
		t := accentStyle.Render(e.task.ID) + "  " + statusStyle(e.task.Status).Render(e.task.Status)
		if g := gatesCell(*e.task); g != "" {
			t += "  " + dimStyle.Render(g)
		}
		if e.task.Drift != "" {
			t += "  " + driftStyle.Render("⚠ "+e.task.Drift)
		}
		return t
	}
}

// selectionKey identifies what the preview should show for the selection.
func (m Model) selectionKey() string {
	e, ok := m.selectedEntry()
	switch {
	case !ok:
		return ""
	case e.track != nil:
		return "track:" + e.track.Path
	default:
		return "task:" + e.task.ID
	}
}

// syncPreview re-renders the preview when the selection changed (or force),
// keeping the scroll offset while the subject stays the same.
func (m Model) syncPreview(force bool) Model {
	key := m.selectionKey()
	if key == m.previewKey && !force {
		return m
	}
	off := 0
	if key == m.previewKey {
		off = m.preview.YOffset
	}
	m.preview.SetContent(m.previewBody())
	m.preview.SetYOffset(off)
	m.previewKey = key
	return m.settleAt(m.preview.YOffset)
}

// previewBody builds the preview content from the snapshot alone: the
// glamour-rendered task body, or a track's summary card.
func (m Model) previewBody() string {
	w := max(m.preview.Width, 1)
	e, ok := m.selectedEntry()
	switch {
	case !ok:
		return dimStyle.Render("nothing selected")
	case e.track != nil:
		return m.trackCard(*e.track, w)
	case e.task.Path == "":
		return dimStyle.Render("no file — the board row points nowhere")
	default:
		return m.taskBody(*e.task, w)
	}
}

// taskBody returns the task's markdown rendered at the preview width,
// cached per id until the next re-scan or resize.
func (m Model) taskBody(r Row, w int) string {
	if s, ok := m.mdCache[r.ID]; ok {
		return s
	}
	s := renderMarkdown(r.Content, w, m.theme)
	m.mdCache[r.ID] = s
	return s
}

// trackCard summarizes a selected track: totals, per-status counts, its
// distribution bar, sections with row counts, and the subtree drift count.
func (m Model) trackCard(s Sub, w int) string {
	lines := []string{
		dimStyle.Render(fmt.Sprintf("%d tasks · %d done", s.Total, s.Done)),
		rollupOrEmpty(s.Counts),
		"",
		statusBar(s.Counts, min(w, 40)),
		"",
	}
	if b, ok := m.snap.Boards[s.Path]; ok && len(b.Sections) > 0 {
		lines = append(lines, sectionStyle.Render("Sections"))
		for _, sec := range b.Sections {
			lines = append(lines, " "+sec.Name+"  "+dimStyle.Render(strconv.Itoa(len(sec.Rows))))
		}
		lines = append(lines, "")
	}
	if n := m.subtreeDrift(s.Path); n > 0 {
		lines = append(lines, driftStyle.Render(fmt.Sprintf("⚠ %d drift", n)))
	} else {
		lines = append(lines, dimStyle.Render("no drift"))
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	return strings.Join(lines, "\n")
}

// subtreeDrift tallies drift-flagged rows on the track at path and every
// track below it.
func (m Model) subtreeDrift(path string) int {
	n := 0
	for p, b := range m.snap.Boards {
		if within(p, path) {
			n += driftCount(b)
		}
	}
	return n
}

// rollupOrEmpty renders the per-status rollup, or a dim "empty" when the
// subtree holds no tasks.
func rollupOrEmpty(counts map[string]int) string {
	if r := statusRollup(counts); r != "" {
		return r
	}
	return dimStyle.Render("empty")
}

// statusRollup renders per-status counts in rollupOrder, each colored with its
// status color, zero counts omitted, joined by a dim middot; "" when empty.
func statusRollup(counts map[string]int) string {
	var parts []string
	for _, st := range rollupOrder {
		if n := counts[st]; n > 0 {
			parts = append(parts, statusStyle(st).Render(fmt.Sprintf("%d %s", n, st)))
		}
	}
	return strings.Join(parts, dimStyle.Render(" · "))
}

// breadcrumb joins the H1 titles from the root down to the track on screen.
func (m Model) breadcrumb() string {
	var parts []string
	p := m.path
	for {
		b, ok := m.snap.Boards[p]
		if !ok {
			break
		}
		parts = append([]string{b.Title}, parts...)
		if b.Parent == "" {
			break
		}
		p = b.Parent
	}
	if len(parts) == 0 {
		return m.path
	}
	return strings.Join(parts, " › ")
}

// renderMarkdown renders a task file for the preview: frontmatter stripped
// (the title line already shows it), glamour sized to width, styled per the
// configured theme (auto detects the terminal). On any renderer error the
// raw markdown shows instead.
func renderMarkdown(content []byte, width int, theme string) string {
	content = task.Body(content)
	opts := []glamour.TermRendererOption{glamour.WithWordWrap(max(width-2, 20))}
	switch theme {
	case "dark", "light":
		opts = append(opts, glamour.WithStandardStyle(theme))
	default:
		opts = append(opts, glamour.WithAutoStyle())
	}
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return string(content)
	}
	out, err := r.Render(string(content))
	if err != nil {
		return string(content)
	}
	return out
}

// cursorMark is the two-column selection gutter.
func cursorMark(selected bool) string {
	if selected {
		return accentStyle.Bold(true).Render("❯") + " "
	}
	return "  "
}

// gatesCell renders gate progress, blank when a task declares no gates.
func gatesCell(r Row) string {
	if r.GatesTotal == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", r.GatesDone, r.GatesTotal)
}

// pad truncates s to w cells with an ellipsis and pads it back to exactly w.
func pad(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if gap := w - lipgloss.Width(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

// driftCount tallies the board's rows carrying a drift flag.
func driftCount(b Board) int {
	n := 0
	for _, s := range b.Sections {
		for _, r := range s.Rows {
			if r.Drift != "" {
				n++
			}
		}
	}
	return n
}

// maxSubNameWidth sizes the track name column.
func maxSubNameWidth(subs []Sub) int {
	w := 0
	for _, s := range subs {
		w = max(w, lipgloss.Width(s.Name))
	}
	return w
}

// maxIDWidth sizes the id column across every section.
func maxIDWidth(sections []Section) int {
	w := 0
	for _, s := range sections {
		for _, r := range s.Rows {
			w = max(w, lipgloss.Width(r.ID))
		}
	}
	return w
}

// maxDriftWidth sizes the drift-badge column; 0 when the board has none.
func maxDriftWidth(sections []Section) int {
	w := 0
	for _, s := range sections {
		for _, r := range s.Rows {
			if r.Drift != "" {
				w = max(w, lipgloss.Width("⚠ "+r.Drift))
			}
		}
	}
	return w
}
