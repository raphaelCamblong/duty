package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/charmbracelet/x/ansi"

	"github.com/raphaelCamblong/duty/internal/task"
)

const (
	statusColWidth   = len(task.StatusInProgress)
	gatesColWidth    = 5
	minTitleWidth    = 8
	twoPanelMinWidth = 80
	minLeftWidth     = 30
	minBarWidth      = 8
	trackBarWidth    = 14
	// narrowCols is the terminal width below which the list drops the gate
	// column, keeping the always-on age column room; the preview header keeps
	// both regardless.
	narrowCols = 100
)

// rollupOrder is the status sequence for track rollups and summaries: active
// work first, then queued, blocked, parked, and finished — matching the header
// bar's colors.
var rollupOrder = []string{task.StatusInProgress, task.StatusTodo, task.StatusBlocked, task.StatusBacklog, task.StatusDone}

const (
	zoneList        = "panel-list"
	zonePreview     = "panel-preview"
	crumbZonePrefix = "crumb:"
)

func crumbZone(path string) string { return crumbZonePrefix + path }

// View renders the current frame: header, the body (a full-width browsing
// list, the open split, or the narrow full-screen preview), and the help
// footer. The zone manager's Scan registers the hit-zones and strips markers.
// The returned view carries the alt-screen and cell-motion mouse modes that
// were program options under Bubble Tea v1.
func (m Model) View() tea.View {
	width, _ := m.dims()
	var body string
	switch {
	case m.split():
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.leftPanel(), m.rightPanel())
	case m.previewOpen:
		title := ansi.Truncate(" "+m.previewTitleText, max(width-1, 1), "…")
		body = lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	default:
		body = m.leftPanel()
	}
	frame := lipgloss.JoinVertical(lipgloss.Left, m.headerView(width), body, m.footerView(width))
	view := tea.NewView(m.zones.Scan(frame))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

func (m Model) layout() Model {
	width, height := m.dims()
	bodyH := max(height-lipgloss.Height(m.headerView(width))-lipgloss.Height(m.footerView(width)), 3)
	switch {
	case m.split():
		lw := leftWidth(width)
		m.list.SetSize(lw-4, bodyH-2)
		m.preview.SetWidth(max(width-lw-4, 1))
		m.preview.SetHeight(max(bodyH-3, 1))
	case m.previewOpen:
		m.list.SetSize(width-4, bodyH-2)
		m.preview.SetWidth(max(width-2, 1))
		m.preview.SetHeight(max(bodyH-1, 1))
	default:
		m.list.SetSize(width-4, bodyH-2)
	}
	return m.renderPreview(false)
}

func leftWidth(width int) int {
	return max(width*38/100, minLeftWidth)
}

// leftPanel is the entry list in its focus-colored border, a full-panel
// mouse zone. A board with no rows or tracks shows a centered dim hint in
// place of the list; a filter that matches nothing falls through to the
// list's own styled no-items state.
func (m Model) leftPanel() string {
	lw, lh := m.list.Width(), m.list.Height()
	box := m.theme.panelBox(m.focus == focusList)
	box = box.Width(lw + box.GetHorizontalFrameSize()).Height(lh + box.GetVerticalFrameSize())
	inner := m.list.View()
	switch {
	case !anySelectable(m.list.Items()):
		inner = lipgloss.Place(lw, lh, lipgloss.Center, lipgloss.Center, m.emptyHint())
	case !anySelectable(m.list.VisibleItems()):
		inner = m.list.Styles.NoItems.Render("No matches")
	}
	return m.zones.Mark(zoneList, box.Render(inner))
}

func (m Model) emptyHint() string {
	if board, ok := m.snap.Boards[m.path]; ok && board.ArchivedSubtree > 0 {
		return m.theme.dim().Render("all work archived — press a to browse the record")
	}
	if m.path == "." {
		return m.theme.dim().Render(`empty tree — duty create task "…" to begin`)
	}
	return m.theme.dim().Render(`no tasks yet — duty create task "…"`)
}

func (m Model) rightPanel() string {
	title := ansi.Truncate(m.previewTitleText, m.preview.Width(), "…")
	box := m.theme.panelBox(m.focus == focusPreview)
	box = box.Width(m.preview.Width() + box.GetHorizontalFrameSize()).Height(m.preview.Height() + 1 + box.GetVerticalFrameSize())
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	return m.zones.Mark(zonePreview, box.Render(content))
}

func (m Model) headerView(width int) string {
	inner := max(width-4, 1)
	board, _ := m.board()
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		ansi.Truncate(m.breadcrumb(), inner, "…"),
		m.theme.stateLine(board, inner),
	)
	box := m.theme.focusBox()
	return box.Width(inner + box.GetHorizontalFrameSize()).Render(content)
}

func (t Theme) stateLine(board Board, width int) string {
	rollup := t.rollupOrEmpty(board.Counts)
	barW := width - lipgloss.Width(rollup) - 2
	if barW < minBarWidth {
		return ansi.Truncate(rollup, width, "…")
	}
	return rollup + "  " + t.statusBar(board.Counts, barW)
}

func (t Theme) statusBar(counts map[string]int, width int) string {
	if totalCount(counts) == 0 {
		return t.dim().Render(strings.Repeat("╌", width))
	}
	bar := barchart.New(
		width, 1,
		barchart.WithHorizontalBars(),
		barchart.WithNoAxis(),
		barchart.WithBarWidth(1),
		barchart.WithDataSet([]barchart.BarData{t.barData(counts)}),
	)
	bar.Draw()
	return bar.View()
}

func (t Theme) barData(counts map[string]int) barchart.BarData {
	values := make([]barchart.BarValue, 0, len(rollupOrder))
	for _, status := range rollupOrder {
		color := t.statusColor(status)
		values = append(values, barchart.BarValue{
			Name:  status,
			Value: float64(counts[status]),
			Style: lipgloss.NewStyle().Foreground(color).Background(color),
		})
	}
	return barchart.BarData{Values: values}
}

func (m Model) footerView(width int) string {
	if m.scanErr == "" {
		return m.helpView(width)
	}
	err := " " + m.theme.alert().Render(ansi.Truncate(m.scanErr, max(width-2, 1), "…"))
	return lipgloss.JoinVertical(lipgloss.Left, err, m.helpView(width))
}

// helpView renders the bubbles/help hint bar (short, or the full grid after
// "?"), truncated per line so a hint set wider than the terminal degrades to
// an ellipsis instead of overflowing the frame.
func (m Model) helpView(width int) string {
	inner := max(width-1, 1)
	help := m.help
	help.SetWidth(inner)
	lines := strings.Split(help.View(m.keys), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], inner, "…")
	}
	return " " + strings.Join(lines, "\n")
}

func (m Model) previewTitle() string {
	switch m.previewKind {
	case previewTrack:
		if sub, ok := m.findSub(m.previewArg); ok {
			return m.theme.accent().Render(sub.Name) + "  " + lipgloss.NewStyle().Bold(true).Render(sub.Title)
		}
	case previewTask:
		if row, board, ok := m.findRowBoard(m.previewArg); ok {
			return m.theme.taskHeader(row, board.Title, m.headerAge(row), m.spinnerGlyph())
		}
	}
	return m.theme.dim().Render("gone")
}

func (m Model) headerAge(row Row) string {
	if !m.showAge {
		return ""
	}
	return ageCell(row, time.Now())
}

// taskHeader joins a task's identity into the preview's pinned line: id ·
// status · gates n/m · track title, then blocked-by ids (met ones struck
// through), any drift badge, and the relative age trailing dim. age trails last
// so a narrow header truncates it before the blocked-by link; age is "" when
// the age column is hidden.
func (t Theme) taskHeader(row Row, track, age, glyph string) string {
	status := t.statusStyle(row.Status).Render(row.Status)
	if inProgress(row) && glyph != "" {
		status += " " + glyph
	}
	parts := []string{t.accent().Render(row.ID), status}
	if who := claimerTag(row); who != "" {
		parts = append(parts, t.dim().Render(who))
	}
	if gates := gatesCell(row); gates != "" {
		parts = append(parts, t.dim().Render(gates))
	}
	if track != "" {
		parts = append(parts, t.dim().Render(track))
	}
	line := strings.Join(parts, t.dim().Render(" · "))
	if len(row.BlockedBy) > 0 {
		line += "  " + t.blockedByCell(row)
	}
	if row.Drift != "" {
		line += "  " + t.alert().Render("⚠ "+row.Drift)
	}
	if age != "" {
		line += "  " + t.dim().Render(age)
	}
	return line
}

// blockedByCell renders the preview header's dim "blocked-by <ids>" segment,
// striking through every met prerequisite so a reader sees at a glance which
// ids still block the task. r.Waits (from the scan) names the unmet ones; the
// label and first-listed unmet id stay contiguous, base-styled, so the segment
// reads plainly when nothing is met yet.
func (t Theme) blockedByCell(row Row) string {
	label := "blocked-by "
	text := label + strings.Join(row.BlockedBy, ", ")
	met := metRunes(len(label), row.BlockedBy, row.Waits)
	if len(met) == 0 {
		return t.dim().Render(text)
	}
	return lipgloss.StyleRunes(text, met, t.dim().Strikethrough(true), t.dim())
}

// metRunes lists, within a "blocked-by <ids>" string, the rune indices of the
// met prerequisites — those absent from waits. ids and the label are ASCII, so
// rune index equals byte offset.
func metRunes(labelLen int, ids, waits []string) []int {
	unmet := make(map[string]bool, len(waits))
	for _, id := range waits {
		unmet[id] = true
	}
	var runes []int
	pos := labelLen
	for i, id := range ids {
		if i > 0 {
			pos += len(", ")
		}
		if !unmet[id] {
			for offset := 0; offset < len(id); offset++ {
				runes = append(runes, pos+offset)
			}
		}
		pos += len(id)
	}
	return runes
}

// previewContent renders the open subject from the snapshot alone: a task's
// markdown through the shared renderer, or a track's summary card. It returns
// the possibly-updated model because building the renderer mutates it.
func (m Model) previewContent() (Model, string) {
	width := max(m.preview.Width(), 1)
	switch m.previewKind {
	case previewTrack:
		if sub, ok := m.findSub(m.previewArg); ok {
			return m, m.trackCard(sub, width)
		}
		return m, m.theme.dim().Render("track gone")
	case previewTask:
		row, _, ok := m.findRowBoard(m.previewArg)
		switch {
		case !ok:
			return m, m.theme.dim().Render("task gone")
		case row.Path == "":
			return m, m.theme.dim().Render("no file — the board row points nowhere")
		default:
			return m.taskMarkdown(row.Content)
		}
	}
	return m, ""
}

// taskMarkdown renders task content through the one shared glamour renderer,
// built lazily on the first open and rebuilt only when the preview width
// changes; the raw markdown shows on any renderer error.
func (m Model) taskMarkdown(content []byte) (Model, string) {
	wrap := max(m.preview.Width()-2, 20)
	if m.renderer == nil || m.rendererWidth != wrap {
		renderer, err := newRenderer(wrap, m.theme.dark)
		if err != nil {
			return m, string(task.Body(content))
		}
		m.renderer, m.rendererWidth = renderer, wrap
	}
	out, err := m.renderer.Render(string(task.Body(content)))
	if err != nil {
		return m, string(task.Body(content))
	}
	return m, out
}

func (m Model) trackCard(sub Sub, width int) string {
	lines := []string{
		m.theme.dim().Render(fmt.Sprintf("%d tasks · %d done", sub.Total, sub.Done)),
		m.theme.rollupOrEmpty(sub.Counts),
		"",
		m.theme.statusBar(sub.Counts, min(width, 40)),
		"",
	}
	if board, ok := m.snap.Boards[sub.Path]; ok && len(board.Sections) > 0 {
		lines = append(lines, m.theme.section().Render("Sections"))
		for _, sec := range board.Sections {
			lines = append(lines, " "+sec.Name+"  "+m.theme.dim().Render(strconv.Itoa(len(sec.Rows))))
		}
		lines = append(lines, "")
	}
	if drift := m.subtreeDrift(sub.Path); drift > 0 {
		lines = append(lines, m.theme.alert().Render(fmt.Sprintf("⚠ %d drift", drift)))
	} else {
		lines = append(lines, m.theme.dim().Render("no drift"))
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "…")
	}
	return strings.Join(lines, "\n")
}

func (m Model) subtreeDrift(path string) int {
	count := 0
	for boardPath, board := range m.snap.Boards {
		if within(boardPath, path) {
			count += driftCount(board)
		}
	}
	return count
}

func (t Theme) rollupOrEmpty(counts map[string]int) string {
	if rollup := t.statusRollup(counts); rollup != "" {
		return rollup
	}
	return t.dim().Render("empty")
}

func (t Theme) statusRollup(counts map[string]int) string {
	var parts []string
	for _, st := range rollupOrder {
		if count := counts[st]; count > 0 {
			parts = append(parts, t.statusStyle(st).Render(fmt.Sprintf("%d %s", count, st)))
		}
	}
	return strings.Join(parts, t.dim().Render(" · "))
}

func totalCount(counts map[string]int) int {
	total := 0
	for _, st := range rollupOrder {
		total += counts[st]
	}
	return total
}

// BarCells splits width cells across the non-zero statuses in counts,
// proportional to each status's share, every non-zero status guaranteed at
// least one cell and leftover cells given to the largest remainders; an empty
// subtree yields nil. width must be at least the number of non-zero statuses.
func BarCells(counts map[string]int, width int) map[string]int {
	total := totalCount(counts)
	if total == 0 {
		return nil
	}
	var active []string
	for _, st := range rollupOrder {
		if counts[st] > 0 {
			active = append(active, st)
		}
	}
	cells := make(map[string]int, len(active))
	rem := make(map[string]float64, len(active))
	spare := width - len(active)
	sum := 0
	for _, st := range active {
		exact := float64(counts[st]) / float64(total) * float64(spare)
		count := 1 + int(exact)
		cells[st] = count
		rem[st] = exact - float64(int(exact))
		sum += count
	}
	for sum < width {
		st := maxRemainder(active, rem)
		cells[st]++
		rem[st]--
		sum++
	}
	return cells
}

func maxRemainder(active []string, rem map[string]float64) string {
	best := active[0]
	for _, st := range active[1:] {
		if rem[st] > rem[best] {
			best = st
		}
	}
	return best
}

func (t Theme) trackBar(counts map[string]int, width int) string {
	cells := BarCells(counts, width)
	if cells == nil {
		return ""
	}
	var builder strings.Builder
	for _, st := range rollupOrder {
		if count := cells[st]; count > 0 {
			builder.WriteString(lipgloss.NewStyle().Foreground(t.statusColor(st)).Render(strings.Repeat("█", count)))
		}
	}
	return builder.String()
}

// trackRightWidth is the fixed cell width of a track row's trailing state
// column: the bar, a two-cell gap, and the right-aligned total count. Both
// trackBarCell and trackLine's title-pad reserve exactly this width.
func trackRightWidth(countW int) int { return trackBarWidth + 2 + countW }

func (t Theme) trackBarCell(counts map[string]int, countW int) string {
	bar := t.trackBar(counts, trackBarWidth)
	if bar == "" {
		return pad(t.dim().Render("empty"), trackRightWidth(countW))
	}
	return bar + "  " + t.dim().Render(fmt.Sprintf("%*d", countW, totalCount(counts)))
}

// breadcrumb joins the H1 titles from the root down to the track on screen,
// each segment a BubbleZone the mouse can click to jump to that ancestor.
func (m Model) breadcrumb() string {
	chain := m.crumbChain()
	if len(chain) == 0 {
		return m.path
	}
	parts := make([]string, len(chain))
	for i, path := range chain {
		seg := m.theme.crumb().Render(m.snap.Boards[path].Title)
		parts[i] = m.zones.Mark(crumbZone(path), seg)
	}
	return strings.Join(parts, m.theme.dim().Render(" › "))
}

// crumbChain lists the board paths from the root down to the track on screen,
// the order the breadcrumb reads left to right.
func (m Model) crumbChain() []string {
	var chain []string
	path := m.path
	for {
		board, ok := m.snap.Boards[path]
		if !ok {
			break
		}
		chain = append([]string{path}, chain...)
		if board.Parent == "" {
			break
		}
		path = board.Parent
	}
	return chain
}

// newRenderer builds the single glamour renderer used program-wide, at a
// fixed word-wrap and a concrete style. The theme is resolved to dark/light
// before the program starts (run.go), so no terminal query fires here.
func newRenderer(wrap int, dark bool) (*glamour.TermRenderer, error) {
	style := "dark"
	if !dark {
		style = "light"
	}
	return glamour.NewTermRenderer(
		glamour.WithWordWrap(wrap),
		glamour.WithStandardStyle(style),
	)
}

func gatesCell(row Row) string {
	if row.GatesTotal == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", row.GatesDone, row.GatesTotal)
}

func inProgress(row Row) bool { return row.Status == task.StatusInProgress }

func claimerTag(row Row) string {
	if inProgress(row) {
		return row.ClaimedBy
	}
	return ""
}

func waitsTag(row Row) string {
	if len(row.Waits) == 0 {
		return ""
	}
	return "waits " + strings.Join(row.Waits, ",")
}

func pad(text string, width int) string {
	text = ansi.Truncate(text, width, "…")
	if gap := width - lipgloss.Width(text); gap > 0 {
		text += strings.Repeat(" ", gap)
	}
	return text
}

func driftCount(board Board) int {
	count := 0
	for _, section := range board.Sections {
		for _, row := range section.Rows {
			if row.Drift != "" {
				count++
			}
		}
	}
	return count
}

func maxSubNameWidth(subs []Sub) int {
	width := 0
	for _, sub := range subs {
		width = max(width, lipgloss.Width(sub.Name))
	}
	return width
}

// maxSubCountWidth sizes the track rows' dim total-count column, measured over
// the non-empty subtrees (an empty subtree shows "empty", carries no count).
func maxSubCountWidth(subs []Sub) int {
	width := 0
	for _, sub := range subs {
		if count := totalCount(sub.Counts); count > 0 {
			width = max(width, len(strconv.Itoa(count)))
		}
	}
	return width
}

func maxIDWidth(sections []Section) int {
	width := 0
	for _, section := range sections {
		for _, row := range section.Rows {
			width = max(width, lipgloss.Width(row.ID))
		}
	}
	return width
}

func maxDriftWidth(sections []Section) int {
	width := 0
	for _, section := range sections {
		for _, row := range section.Rows {
			if row.Drift != "" {
				width = max(width, lipgloss.Width("⚠ "+row.Drift))
			}
		}
	}
	return width
}
