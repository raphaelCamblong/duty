package tui

import (
	"fmt"
	"math"
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
	// previewLines caps the selected task's Goal preview; extra text is
	// ellipsis-truncated onto the last line.
	previewLines = 4
	// minBodyLines is the fewest task rows the preview pane will leave
	// visible; below it the pane auto-hides on short terminals.
	minBodyLines = 3
)

// rollupOrder is the status sequence for track rollups and previews: active
// work first, then queued, blocked, and finished — matching the header bar's
// colors.
var rollupOrder = []string{task.StatusInProgress, task.StatusTodo, task.StatusBlocked, task.StatusDone}

var (
	colAccent = lipgloss.AdaptiveColor{Light: "62", Dark: "111"}
	colDim    = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	colYellow = lipgloss.AdaptiveColor{Light: "136", Dark: "220"}
	colRed    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	colGreen  = lipgloss.AdaptiveColor{Light: "28", Dark: "78"}

	headerBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
	previewBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colDim).Padding(0, 1)
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

// statusColor is the fill for a status's segment of the header distribution
// bar, matching statusStyle's foregrounds.
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

// View renders the current frame: the open detail, or the board. The zone
// manager's Scan registers the row hit-zones and strips their markers.
func (m Model) View() string {
	if m.detailOpen {
		return m.zones.Scan(m.detailView())
	}
	return m.zones.Scan(m.boardView())
}

// boardGeom is one frame's board layout, shared by the view and the mouse
// hit-test so both agree on where every row sits.
type boardGeom struct {
	header, footer string
	preview        string
	lines          []string
	lineItem       []int
	selLine        int
	headerH        int
	footerH        int
	previewH       int
	visible        int
	total          int
	maxTop         int
	top            int
}

// geom computes the board layout at the current size: header, footer, body
// lines with their item indices, and the scroll window derived from the
// spring position.
func (m Model) geom() boardGeom {
	w, h := m.dims()
	header := m.boardHeader(w)
	footer := m.boardFooter(w)
	lines, lineItem, selLine := m.bodyLines(w)
	g := boardGeom{
		header: header, footer: footer,
		lines: lines, lineItem: lineItem, selLine: selLine,
		headerH: lipgloss.Height(header), footerH: lipgloss.Height(footer),
		total: len(lines),
	}
	avail := h - g.headerH - g.footerH
	if preview := m.preview(w); preview != "" && avail-lipgloss.Height(preview) >= minBodyLines {
		g.preview, g.previewH = preview, lipgloss.Height(preview)
	}
	g.visible = max(avail-g.previewH, 1)
	g.maxTop = max(g.total-g.visible, 0)
	g.top = clamp(int(math.Round(m.scroll)), 0, g.maxTop)
	return g
}

// boardView lays out header, the scroll window of the board's lines, and the
// footer, padded so the footer sits on the bottom row.
func (m Model) boardView() string {
	g := m.geom()
	body := windowLines(g.lines, g.top, g.visible)
	parts := []string{g.header, strings.Join(body, "\n")}
	if g.preview != "" {
		parts = append(parts, g.preview)
	}
	parts = append(parts, g.footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// detailView lays out the task header, the glamour-rendered markdown in its
// viewport, and a scroll-position footer with key hints.
func (m Model) detailView() string {
	w, _ := m.dims()
	scroll := " " + dimStyle.Render(fmt.Sprintf("%3.0f%%", m.detailVP.ScrollPercent()*100))
	footer := lipgloss.JoinVertical(lipgloss.Left, scroll, m.helpView(w))
	return lipgloss.JoinVertical(lipgloss.Left, m.detailHeader(m.detailRow, w), m.detailVP.View(), footer)
}

// detailHeader is the boxed breadcrumb + status/gates line above a task's
// rendered markdown.
func (m Model) detailHeader(r Row, w int) string {
	crumb := crumbStyle.Render(m.breadcrumb()+" › "+r.ID) + "  " +
		statusStyle(r.Status).Render(r.Status) + "  " +
		dimStyle.Render(gatesCell(r))
	return headerBox.Width(max(w-2, 1)).Render(ansi.Truncate(crumb, max(w-4, 1), "…"))
}

// boardHeader is the rounded box holding the breadcrumb and the one-line
// status-distribution bar (§8).
func (m Model) boardHeader(w int) string {
	inner := max(w-4, 1)
	content := lipgloss.JoinVertical(lipgloss.Left,
		ansi.Truncate(crumbStyle.Render(m.breadcrumb()), inner, "…"),
		m.statusBar(inner),
	)
	return headerBox.Width(max(w-2, 1)).Render(content)
}

// statusBar renders the current board's task statuses as one horizontal
// ntcharts bar w cells wide; an empty board shows a faint rule.
func (m Model) statusBar(w int) string {
	b, ok := m.board()
	counts := statusCounts(b)
	total := counts[task.StatusTodo] + counts[task.StatusInProgress] + counts[task.StatusBlocked] + counts[task.StatusDone]
	if !ok || total == 0 {
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

// statusCounts tallies the board's direct task rows by status.
func statusCounts(b Board) map[string]int {
	c := map[string]int{}
	for _, s := range b.Sections {
		for _, r := range s.Rows {
			c[r.Status]++
		}
	}
	return c
}

// boardFooter is the subtle status line — subtree counts and drift total, or
// the last scan error — above the "?" key hints.
func (m Model) boardFooter(w int) string {
	return lipgloss.JoinVertical(lipgloss.Left, m.statusLine(w), m.helpView(w))
}

// statusLine summarizes the board: done/total, drift, or the last scan error.
func (m Model) statusLine(w int) string {
	if m.scanErr != "" {
		return " " + errStyle.Render(ansi.Truncate(m.scanErr, max(w-2, 1), "…"))
	}
	b, ok := m.board()
	if !ok {
		return " " + dimStyle.Render("no board")
	}
	s := fmt.Sprintf("%d/%d done", b.Done, b.Total)
	if n := driftCount(b); n > 0 {
		s += fmt.Sprintf(" · %d drift", n)
	}
	return " " + dimStyle.Render(s)
}

// helpView renders the bubbles/help hint bar (short, or the full grid after
// "?"), sized to the terminal.
func (m Model) helpView(w int) string {
	h := m.help
	h.Width = w
	return " " + h.View(m.keys)
}

// bodyLines renders the board's lines — sub-boards, then each section header
// and its rows, each row wrapped as a zone — and returns, per line, the
// selectable item index (-1 for blanks and section labels) plus the selected
// line.
func (m Model) bodyLines(w int) (lines []string, lineItem []int, sel int) {
	b, ok := m.board()
	if !ok {
		return []string{" " + dimStyle.Render("no boards found")}, []int{-1}, 0
	}
	add := func(line string, item int) {
		lines = append(lines, line)
		lineItem = append(lineItem, item)
	}
	cursor := m.cursor()
	idx := 0
	nameW := maxSubNameWidth(b.Subs)
	for i := range b.Subs {
		if idx == cursor {
			sel = len(lines)
		}
		add(m.zone(idx, subLine(b.Subs[i], nameW, idx == cursor, w)), idx)
		idx++
	}
	idW := maxIDWidth(b.Sections)
	driftW := maxDriftWidth(b.Sections)
	for _, s := range b.Sections {
		add("", -1)
		add(" "+sectionStyle.Render(s.Name), -1)
		for ri := range s.Rows {
			if idx == cursor {
				sel = len(lines)
			}
			add(m.zone(idx, rowLine(s.Rows[ri], idW, driftW, idx == cursor, w)), idx)
			idx++
		}
	}
	if idx == 0 {
		add(" "+dimStyle.Render("no open tasks"), -1)
	}
	return lines, lineItem, sel
}

// zone wraps a rendered row in a bubblezone marker so it is a clickable
// hit-zone in the terminal.
func (m Model) zone(idx int, line string) string {
	return m.zones.Mark(rowZoneID(idx), line)
}

// rowZoneID is the stable zone name for the item at idx on the current board.
func rowZoneID(idx int) string {
	return fmt.Sprintf("row-%d", idx)
}

// windowLines slices lines to the visible window, padding short pages so the
// footer stays pinned to the bottom.
func windowLines(lines []string, top, visible int) []string {
	out := make([]string, 0, visible)
	for i := 0; i < visible; i++ {
		if top+i < len(lines) {
			out = append(out, lines[top+i])
			continue
		}
		out = append(out, "")
	}
	return out
}

// subLine renders one sub-board: name, title, and a per-status rollup of its
// subtree, each count in its status color, zero-count statuses omitted.
func subLine(s Sub, nameW int, selected bool, w int) string {
	title := s.Title
	if selected {
		title = selStyle.Render(title)
	}
	line := cursorMark(selected) +
		accentStyle.Render(pad(s.Name, nameW)) + "  " +
		title + "  " +
		rollupOrEmpty(s.Counts)
	return ansi.Truncate(line, w, "…")
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

// preview builds the bottom pane for the current selection — a task's Goal or
// a track's per-status summary, in a dim rounded box; "" when nothing is
// selected (an empty board).
func (m Model) preview(w int) string {
	it, ok := m.selected()
	if !ok {
		return ""
	}
	inner := max(w-4, 1)
	content := ""
	if it.sub != nil {
		content = trackPreview(*it.sub, inner)
	} else {
		content = goalPreview(it.row.Goal, inner)
	}
	return previewBox.Width(max(w-2, 1)).Render(content)
}

// goalPreview flattens and word-wraps a task's Goal to at most previewLines
// dim lines, ellipsis-truncating the overflow; a task with no Goal reads
// "(no goal)".
func goalPreview(goal string, w int) string {
	flat := strings.Join(strings.Fields(goal), " ")
	if flat == "" {
		return dimStyle.Render("(no goal)")
	}
	lines := strings.Split(ansi.Wordwrap(flat, w, ""), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	if len(lines) > previewLines {
		lines = lines[:previewLines]
		last := len(lines) - 1
		lines[last] = ansi.Truncate(lines[last], max(w-1, 1), "") + "…"
	}
	return dimStyle.Render(strings.Join(lines, "\n"))
}

// trackPreview summarizes a selected sub-board: its name and title, then a
// total and the per-status rollup.
func trackPreview(s Sub, w int) string {
	head := accentStyle.Render(s.Name) + "  " + s.Title
	summary := dimStyle.Render(fmt.Sprintf("%d total", s.Total)) + dimStyle.Render(" · ") + rollupOrEmpty(s.Counts)
	return ansi.Truncate(head, w, "…") + "\n" + ansi.Truncate(summary, w, "…")
}

// rowLine renders one task: id, title, colored status, gate progress, drift
// badge. driftW is the board's widest badge, so the title column yields room
// for badges to stay visible and aligned.
func rowLine(r Row, idW, driftW int, selected bool, w int) string {
	fixed := 2 + idW + 2 + 2 + statusColWidth + 2 + gatesColWidth
	if driftW > 0 {
		fixed += 2 + driftW
	}
	title := pad(r.Title, max(w-fixed, minTitleWidth))
	if selected {
		title = selStyle.Render(title)
	}
	line := cursorMark(selected) +
		accentStyle.Render(pad(r.ID, idW)) + "  " +
		title + "  " +
		statusStyle(r.Status).Render(pad(r.Status, statusColWidth)) + "  " +
		dimStyle.Render(pad(gatesCell(r), gatesColWidth))
	if r.Drift != "" {
		line += "  " + driftStyle.Render(pad("⚠ "+r.Drift, driftW))
	}
	return ansi.Truncate(line, w, "…")
}

// breadcrumb joins the H1 titles from the root down to the board on screen.
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

// renderMarkdown renders a task file for the detail view: frontmatter
// stripped (the header line already shows it), glamour sized to width,
// styled per the configured theme (auto detects the terminal). On any
// renderer error the raw markdown shows instead.
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

// maxSubNameWidth sizes the sub-board name column.
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
