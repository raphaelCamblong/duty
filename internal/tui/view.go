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
	// narrowCols is the terminal width below which the list drops the gate
	// column, keeping the always-on age column room; the preview header keeps
	// both regardless.
	narrowCols = 100
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
	box := m.theme.panelBox(m.focus == focusList).Width(cw).Height(ch)
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
		return m.theme.dim().Render(`empty tree — duty create task "…" to begin`)
	}
	return m.theme.dim().Render(`no tasks yet — duty create task "…"`)
}

// rightPanel is the preview — pinned title line over the viewport — in its
// focus-colored border, a full-panel mouse zone.
func (m Model) rightPanel() string {
	title := ansi.Truncate(m.previewTitleText, m.preview.Width, "…")
	box := m.theme.panelBox(m.focus == focusPreview).Width(m.preview.Width + 2).Height(m.preview.Height + 1)
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
		m.theme.stateLine(b, inner),
	)
	return m.theme.focusBox().Width(max(w-2, 1)).Render(content)
}

// stateLine renders a board's subtree per-status counts in status colors,
// with the ntcharts distribution bar filling the rest of the line.
func (t Theme) stateLine(b Board, w int) string {
	rollup := t.rollupOrEmpty(b.Counts)
	barW := w - lipgloss.Width(rollup) - 2
	if barW < minBarWidth {
		return ansi.Truncate(rollup, w, "…")
	}
	return rollup + "  " + t.statusBar(b.Counts, barW)
}

// statusBar renders per-status counts as one horizontal ntcharts bar w cells
// wide; no tasks shows a faint rule.
func (t Theme) statusBar(counts map[string]int, w int) string {
	if totalCount(counts) == 0 {
		return t.dim().Render(strings.Repeat("╌", w))
	}
	bar := barchart.New(
		w, 1,
		barchart.WithHorizontalBars(),
		barchart.WithNoAxis(),
		barchart.WithBarWidth(1),
		barchart.WithDataSet([]barchart.BarData{t.barData(counts)}),
	)
	bar.Draw()
	return bar.View()
}

// barData turns status counts into one stacked horizontal bar, its segments in
// rollupOrder so the header bar and the inline track bars agree on screen.
func (t Theme) barData(counts map[string]int) barchart.BarData {
	values := make([]barchart.BarValue, 0, len(rollupOrder))
	for _, status := range rollupOrder {
		c := t.statusColor(status)
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
	err := " " + m.theme.alert().Render(ansi.Truncate(m.scanErr, max(w-2, 1), "…"))
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
			return m.theme.accent().Render(s.Name) + "  " + lipgloss.NewStyle().Bold(true).Render(s.Title)
		}
	case previewTask:
		if r, b, ok := m.findRowBoard(m.previewArg); ok {
			return m.theme.taskHeader(r, b.Title, m.headerAge(r))
		}
	}
	return m.theme.dim().Render("gone")
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
// status · gates n/m · track title, then blocked-by ids (met ones struck
// through), any drift badge, and the relative age trailing dim. age trails last
// so a narrow header truncates it before the blocked-by link; age is "" when
// the age column is hidden.
func (t Theme) taskHeader(r Row, track, age string) string {
	parts := []string{t.accent().Render(r.ID), t.statusStyle(r.Status).Render(r.Status)}
	if who := claimerTag(r); who != "" {
		parts = append(parts, t.dim().Render(who))
	}
	if g := gatesCell(r); g != "" {
		parts = append(parts, t.dim().Render(g))
	}
	if track != "" {
		parts = append(parts, t.dim().Render(track))
	}
	line := strings.Join(parts, t.dim().Render(" · "))
	if len(r.BlockedBy) > 0 {
		line += "  " + t.blockedByCell(r)
	}
	if r.Drift != "" {
		line += "  " + t.alert().Render("⚠ "+r.Drift)
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
func (t Theme) blockedByCell(r Row) string {
	label := "blocked-by "
	s := label + strings.Join(r.BlockedBy, ", ")
	met := metRunes(len(label), r.BlockedBy, r.Waits)
	if len(met) == 0 {
		return t.dim().Render(s)
	}
	return lipgloss.StyleRunes(s, met, t.dim().Strikethrough(true), t.dim())
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
			for k := 0; k < len(id); k++ {
				runes = append(runes, pos+k)
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
	w := max(m.preview.Width, 1)
	switch m.previewKind {
	case previewTrack:
		if s, ok := m.findSub(m.previewArg); ok {
			return m, m.trackCard(s, w)
		}
		return m, m.theme.dim().Render("track gone")
	case previewTask:
		r, _, ok := m.findRowBoard(m.previewArg)
		switch {
		case !ok:
			return m, m.theme.dim().Render("task gone")
		case r.Path == "":
			return m, m.theme.dim().Render("no file — the board row points nowhere")
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
		r, err := newRenderer(wrap, m.mode)
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
		m.theme.dim().Render(fmt.Sprintf("%d tasks · %d done", s.Total, s.Done)),
		m.theme.rollupOrEmpty(s.Counts),
		"",
		m.theme.statusBar(s.Counts, min(w, 40)),
		"",
	}
	if b, ok := m.snap.Boards[s.Path]; ok && len(b.Sections) > 0 {
		lines = append(lines, m.theme.section().Render("Sections"))
		for _, sec := range b.Sections {
			lines = append(lines, " "+sec.Name+"  "+m.theme.dim().Render(strconv.Itoa(len(sec.Rows))))
		}
		lines = append(lines, "")
	}
	if n := m.subtreeDrift(s.Path); n > 0 {
		lines = append(lines, m.theme.alert().Render(fmt.Sprintf("⚠ %d drift", n)))
	} else {
		lines = append(lines, m.theme.dim().Render("no drift"))
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
func (t Theme) rollupOrEmpty(counts map[string]int) string {
	if r := t.statusRollup(counts); r != "" {
		return r
	}
	return t.dim().Render("empty")
}

// statusRollup renders per-status counts in rollupOrder, each colored with its
// status color, zero counts omitted, joined by a dim middot; "" when empty.
func (t Theme) statusRollup(counts map[string]int) string {
	var parts []string
	for _, st := range rollupOrder {
		if n := counts[st]; n > 0 {
			parts = append(parts, t.statusStyle(st).Render(fmt.Sprintf("%d %s", n, st)))
		}
	}
	return strings.Join(parts, t.dim().Render(" · "))
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
func (t Theme) trackBar(counts map[string]int, width int) string {
	cells := BarCells(counts, width)
	if cells == nil {
		return ""
	}
	var b strings.Builder
	for _, st := range rollupOrder {
		if c := cells[st]; c > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(t.statusColor(st)).Render(strings.Repeat("█", c)))
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
	for i, p := range chain {
		seg := m.theme.crumb().Render(m.snap.Boards[p].Title)
		parts[i] = m.zones.Mark(crumbZone(p), seg)
	}
	return strings.Join(parts, m.theme.dim().Render(" › "))
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

// gatesCell renders gate progress, blank when a task declares no gates.
func gatesCell(r Row) string {
	if r.GatesTotal == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", r.GatesDone, r.GatesTotal)
}

// claimerTag is the holder name shown dim beside an in-progress row or preview
// header, "" for any other status or an unclaimed task.
func claimerTag(r Row) string {
	if r.Status == task.StatusInProgress {
		return r.ClaimedBy
	}
	return ""
}

// waitsTag is the dim "waits T-01,T-03" annotation for a row with unmet
// dependencies, "" when every blocked-by prerequisite is already met.
func waitsTag(r Row) string {
	if len(r.Waits) == 0 {
		return ""
	}
	return "waits " + strings.Join(r.Waits, ",")
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
