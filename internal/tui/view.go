package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	// trackBarWidth is the fixed cell width of a track row's inline state bar.
	trackBarWidth = 14
	// ageDefaultCols is the terminal width at or above which the relative-age
	// column shows by default; narrower terminals hide it until toggled.
	ageDefaultCols = 100
)

// rollupOrder is the status sequence for track rollups and summaries: active
// work first, then queued, blocked, and finished — matching the header bar's
// colors.
var rollupOrder = []string{task.StatusInProgress, task.StatusTodo, task.StatusBlocked, task.StatusDone}

// zoneList and zonePreview are the BubbleZone names of the two panels.
const (
	zoneList    = "panel-list"
	zonePreview = "panel-preview"
	// crumbZonePrefix names the per-segment breadcrumb hit-zones.
	crumbZonePrefix = "crumb:"
)

// crumbZone is the stable zone name of the breadcrumb segment for board path.
func crumbZone(path string) string { return crumbZonePrefix + path }

// The duty palette (§8): cream #e1ebaf, peach #e1af7d, bronze #af874b, indigo
// #1e0f37, olive #9baf37 — mapped to accent and status below. The palette skews
// dark, so each AdaptiveColor's Light counterpart is a hand-darkened variant
// (indigo is the light-theme ink) that stays legible on a pale terminal. blocked
// keeps a plain red: the palette carries no alarm color, and blocked must alarm.
var (
	// colAccent tints focused borders, the breadcrumb, the selection, and the
	// header title: cream on dark, indigo ink on light.
	colAccent = lipgloss.AdaptiveColor{Light: "#1e0f37", Dark: "#e1ebaf"}
	// colDim tints chrome — separators, ages, hints, blurred borders — in the
	// terminal's own grays, untouched by the palette.
	colDim = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	// colPeach tints in-progress.
	colPeach = lipgloss.AdaptiveColor{Light: "#a5652f", Dark: "#e1af7d"}
	// colBronze tints todo, the palette's quiet earthy tone.
	colBronze = lipgloss.AdaptiveColor{Light: "#8a6a38", Dark: "#af874b"}
	// colOlive tints done.
	colOlive = lipgloss.AdaptiveColor{Light: "#6f7d27", Dark: "#9baf37"}
	// colRed tints blocked, plus scan errors and drift.
	colRed = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}

	headerBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
	focusedBox   = headerBox
	blurredBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colDim).Padding(0, 1)
	crumbStyle   = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	accentStyle  = lipgloss.NewStyle().Foreground(colAccent)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(colDim)
	dimStyle     = lipgloss.NewStyle().Foreground(colDim)
	errStyle     = lipgloss.NewStyle().Foreground(colRed)
	selStyle     = lipgloss.NewStyle().Bold(true)
	driftStyle   = lipgloss.NewStyle().Foreground(colRed)
)

// statusStyle maps a task status to a foreground style in its color — todo
// bronze, in-progress peach, blocked red, done olive — the palette statusColor
// owns.
func statusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(statusColor(status))
}

// statusColor is the fill for a status's segment of a distribution bar,
// matching statusStyle's foregrounds; an unknown status stays dim.
func statusColor(status string) lipgloss.TerminalColor {
	switch status {
	case task.StatusInProgress:
		return colPeach
	case task.StatusTodo:
		return colBronze
	case task.StatusBlocked:
		return colRed
	case task.StatusDone:
		return colOlive
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

// View renders the current frame: header, the body (a full-width browsing
// list, the open split, or the narrow full-screen preview), and the help
// footer. The zone manager's Scan registers the hit-zones and strips markers.
func (m Model) View() string {
	w, _ := m.dims()
	var body string
	switch {
	case m.split():
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.leftPanel(), m.rightPanel())
	case m.previewOpen:
		title := ansi.Truncate(" "+m.previewTitleText, max(w-1, 1), "…")
		body = lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	default:
		body = m.leftPanel()
	}
	frame := lipgloss.JoinVertical(lipgloss.Left, m.headerView(w), body, m.footerView(w))
	return m.zones.Scan(frame)
}

// layout sizes the list and preview to the current terminal and chrome
// heights, then re-renders the open preview at the new width. Browsing gives
// the list the full width; a narrow open preview takes over the body.
func (m Model) layout() Model {
	w, h := m.dims()
	bodyH := max(h-lipgloss.Height(m.headerView(w))-lipgloss.Height(m.footerView(w)), 3)
	switch {
	case m.split():
		lw := leftWidth(w)
		m.list.SetSize(lw-4, bodyH-2)
		m.preview.Width = max(w-lw-4, 1)
		m.preview.Height = max(bodyH-3, 1)
	case m.previewOpen:
		m.list.SetSize(w-4, bodyH-2)
		m.preview.Width = max(w-2, 1)
		m.preview.Height = max(bodyH-1, 1)
	default:
		m.list.SetSize(w-4, bodyH-2)
	}
	return m.renderPreview(false)
}

// leftWidth is the left panel's total width: ~38% of the terminal, floored.
func leftWidth(w int) int {
	return max(w*38/100, minLeftWidth)
}

// leftPanel is the entry list in its focus-colored border, a full-panel
// mouse zone. A board with no rows or tracks shows a centered dim hint in
// place of the list; a filter that matches nothing falls through to the
// list's own styled no-items state.
func (m Model) leftPanel() string {
	cw, ch := m.list.Width()+2, m.list.Height()
	box := panelBox(m.focus == focusList).Width(cw).Height(ch)
	inner := m.list.View()
	switch {
	case !anySelectable(m.list.Items()):
		inner = lipgloss.Place(cw, ch, lipgloss.Center, lipgloss.Center, m.emptyHint())
	case !anySelectable(m.list.VisibleItems()):
		inner = m.list.Styles.NoItems.Render("No matches")
	}
	return m.zones.Mark(zoneList, box.Render(inner))
}

// emptyHint is the centered dim message for a board with no tracks or tasks:
// a fresh tree names itself, any other empty track nudges toward create.
func (m Model) emptyHint() string {
	if m.path == "." {
		return dimStyle.Render(`empty tree — duty create task "…" to begin`)
	}
	return dimStyle.Render(`no tasks yet — duty create task "…"`)
}

// rightPanel is the preview — pinned title line over the viewport — in its
// focus-colored border, a full-panel mouse zone.
func (m Model) rightPanel() string {
	title := ansi.Truncate(m.previewTitleText, m.preview.Width, "…")
	box := panelBox(m.focus == focusPreview).Width(m.preview.Width + 2).Height(m.preview.Height + 1)
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.preview.View())
	return m.zones.Mark(zonePreview, box.Render(content))
}

// headerView is the rounded box holding the breadcrumb and the current
// track's subtree state: per-status counts plus the distribution bar (§8).
func (m Model) headerView(w int) string {
	inner := max(w-4, 1)
	b, _ := m.board()
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		ansi.Truncate(m.breadcrumb(), inner, "…"),
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
	if totalCount(counts) == 0 {
		return dimStyle.Render(strings.Repeat("╌", w))
	}
	bar := barchart.New(
		w, 1,
		barchart.WithHorizontalBars(),
		barchart.WithNoAxis(),
		barchart.WithBarWidth(1),
		barchart.WithDataSet([]barchart.BarData{barData(counts)}),
	)
	bar.Draw()
	return bar.View()
}

// barData turns status counts into one stacked horizontal bar, its segments in
// rollupOrder so the header bar and the inline track bars agree on screen.
func barData(counts map[string]int) barchart.BarData {
	values := make([]barchart.BarValue, 0, len(rollupOrder))
	for _, status := range rollupOrder {
		c := statusColor(status)
		values = append(values, barchart.BarValue{
			Name:  status,
			Value: float64(counts[status]),
			Style: lipgloss.NewStyle().Foreground(c).Background(c),
		})
	}
	return barchart.BarData{Values: values}
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
// "?"), truncated per line so a hint set wider than the terminal degrades to
// an ellipsis instead of overflowing the frame.
func (m Model) helpView(w int) string {
	inner := max(w-1, 1)
	h := m.help
	h.Width = inner
	lines := strings.Split(h.View(m.keys), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], inner, "…")
	}
	return " " + strings.Join(lines, "\n")
}

// previewTitle is the pinned line above the open preview, resolved from the
// open subject: id, colored status, gates n/m, and track title for a task —
// blocked-by ids and drift appended dim when present; name and title for a
// track.
func (m Model) previewTitle() string {
	switch m.previewKind {
	case previewTrack:
		if s, ok := m.findSub(m.previewArg); ok {
			return accentStyle.Render(s.Name) + "  " + selStyle.Render(s.Title)
		}
	case previewTask:
		if r, b, ok := m.findRowBoard(m.previewArg); ok {
			return taskHeader(r, b.Title, m.headerAge(r))
		}
	}
	return dimStyle.Render("gone")
}

// headerAge is the preview header's relative age, "" when the age column is
// hidden or the row has no file.
func (m Model) headerAge(r Row) string {
	if !m.showAge {
		return ""
	}
	return ageCell(r, time.Now())
}

// taskHeader joins a task's identity into the preview's pinned line: id ·
// status · gates n/m · track title · age, with blocked-by ids and any drift
// badge trailing dim. age is "" when the age column is hidden.
func taskHeader(r Row, track, age string) string {
	parts := []string{accentStyle.Render(r.ID), statusStyle(r.Status).Render(r.Status)}
	if g := gatesCell(r); g != "" {
		parts = append(parts, dimStyle.Render(g))
	}
	if track != "" {
		parts = append(parts, dimStyle.Render(track))
	}
	if age != "" {
		parts = append(parts, dimStyle.Render(age))
	}
	line := strings.Join(parts, dimStyle.Render(" · "))
	if len(r.BlockedBy) > 0 {
		line += "  " + dimStyle.Render("blocked-by "+strings.Join(r.BlockedBy, ", "))
	}
	if r.Drift != "" {
		line += "  " + driftStyle.Render("⚠ "+r.Drift)
	}
	return line
}

// previewContent renders the open subject from the snapshot alone: a task's
// markdown through the shared renderer, or a track's summary card. It returns
// the possibly-updated model because building the renderer mutates it.
func (m Model) previewContent() (Model, string) {
	w := max(m.preview.Width, 1)
	switch m.previewKind {
	case previewTrack:
		if s, ok := m.findSub(m.previewArg); ok {
			return m, m.trackCard(s, w)
		}
		return m, dimStyle.Render("track gone")
	case previewTask:
		r, _, ok := m.findRowBoard(m.previewArg)
		switch {
		case !ok:
			return m, dimStyle.Render("task gone")
		case r.Path == "":
			return m, dimStyle.Render("no file — the board row points nowhere")
		default:
			return m.taskMarkdown(r.Content)
		}
	}
	return m, ""
}

// taskMarkdown renders task content through the one shared glamour renderer,
// built lazily on the first open and rebuilt only when the preview width
// changes; the raw markdown shows on any renderer error.
func (m Model) taskMarkdown(content []byte) (Model, string) {
	wrap := max(m.preview.Width-2, 20)
	if m.renderer == nil || m.rendererWidth != wrap {
		r, err := newRenderer(wrap, m.theme)
		if err != nil {
			return m, string(task.Body(content))
		}
		m.renderer, m.rendererWidth = r, wrap
	}
	out, err := m.renderer.Render(string(task.Body(content)))
	if err != nil {
		return m, string(task.Body(content))
	}
	return m, out
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

// totalCount sums a status→count map over the lifecycle statuses.
func totalCount(counts map[string]int) int {
	n := 0
	for _, st := range rollupOrder {
		n += counts[st]
	}
	return n
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
		c := 1 + int(exact)
		cells[st] = c
		rem[st] = exact - float64(int(exact))
		sum += c
	}
	for sum < width {
		st := maxRemainder(active, rem)
		cells[st]++
		rem[st]--
		sum++
	}
	return cells
}

// maxRemainder is the status with the largest fractional remainder, ties
// broken by lifecycle order.
func maxRemainder(active []string, rem map[string]float64) string {
	best := active[0]
	for _, st := range active[1:] {
		if rem[st] > rem[best] {
			best = st
		}
	}
	return best
}

// trackBar renders a fixed-width inline status-distribution bar: colored
// block runs proportional to the subtree per-status counts in lifecycle
// order, the header bar's palette, "" when the subtree holds no tasks.
func trackBar(counts map[string]int, width int) string {
	cells := BarCells(counts, width)
	if cells == nil {
		return ""
	}
	var b strings.Builder
	for _, st := range rollupOrder {
		if c := cells[st]; c > 0 {
			b.WriteString(statusStyle(st).Render(strings.Repeat("█", c)))
		}
	}
	return b.String()
}

// trackRightWidth is the fixed cell width of a track row's trailing state
// column: the bar, a two-cell gap, and the right-aligned total count. Both
// trackBarCell and trackLine's title-pad reserve exactly this width.
func trackRightWidth(countW int) int { return trackBarWidth + 2 + countW }

// trackBarCell is a track row's right-aligned state column, a fixed
// trackRightWidth cells: the status-distribution bar and a right-aligned
// dim total count, or a dim "empty" filling the column when the subtree holds
// no tasks.
func trackBarCell(counts map[string]int, countW int) string {
	bar := trackBar(counts, trackBarWidth)
	if bar == "" {
		return pad(dimStyle.Render("empty"), trackRightWidth(countW))
	}
	return bar + "  " + dimStyle.Render(fmt.Sprintf("%*d", countW, totalCount(counts)))
}

// breadcrumb joins the H1 titles from the root down to the track on screen,
// each segment a BubbleZone the mouse can click to jump to that ancestor.
func (m Model) breadcrumb() string {
	chain := m.crumbChain()
	if len(chain) == 0 {
		return m.path
	}
	parts := make([]string, len(chain))
	for i, p := range chain {
		seg := crumbStyle.Render(m.snap.Boards[p].Title)
		parts[i] = m.zones.Mark(crumbZone(p), seg)
	}
	return strings.Join(parts, dimStyle.Render(" › "))
}

// crumbChain lists the board paths from the root down to the track on screen,
// the order the breadcrumb reads left to right.
func (m Model) crumbChain() []string {
	var chain []string
	p := m.path
	for {
		b, ok := m.snap.Boards[p]
		if !ok {
			break
		}
		chain = append([]string{p}, chain...)
		if b.Parent == "" {
			break
		}
		p = b.Parent
	}
	return chain
}

// newRenderer builds the single glamour renderer used program-wide, at a
// fixed word-wrap and a concrete style. The theme is resolved to dark/light
// before the program starts (run.go), so no terminal query fires here.
func newRenderer(wrap int, theme string) (*glamour.TermRenderer, error) {
	style := "dark"
	if theme == "light" {
		style = "light"
	}
	return glamour.NewTermRenderer(
		glamour.WithWordWrap(wrap),
		glamour.WithStandardStyle(style),
	)
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

// maxSubCountWidth sizes the track rows' dim total-count column, measured over
// the non-empty subtrees (an empty subtree shows "empty", carries no count).
func maxSubCountWidth(subs []Sub) int {
	w := 0
	for _, s := range subs {
		if n := totalCount(s.Counts); n > 0 {
			w = max(w, len(strconv.Itoa(n)))
		}
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
