package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/raphaelCamblong/duty/internal/task"
)

// detailChrome is the lines the detail view spends around the viewport:
// a three-line bordered header and a one-line footer.
const detailChrome = 4

const (
	statusColWidth = len(task.StatusInProgress)
	gatesColWidth  = 5
	minTitleWidth  = 8
)

var (
	colAccent = lipgloss.AdaptiveColor{Light: "62", Dark: "111"}
	colDim    = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	colYellow = lipgloss.AdaptiveColor{Light: "136", Dark: "220"}
	colRed    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	colGreen  = lipgloss.AdaptiveColor{Light: "28", Dark: "78"}

	headerBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
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

// View renders the current frame: the open detail, or the board.
func (m Model) View() string {
	if m.detailOpen {
		return m.detailView()
	}
	return m.boardView()
}

// boardView lays out header, a scroll window of the board's lines, and the
// footer, padded so the footer sits on the bottom row.
func (m Model) boardView() string {
	w, h := m.dims()
	header := m.headerLine(m.breadcrumb(), w)
	footer := m.footerLine(w)
	visible := max(h-lipgloss.Height(header)-lipgloss.Height(footer), 1)
	lines, sel := m.bodyLines(w)
	lines = window(lines, sel, visible)
	for len(lines) < visible {
		lines = append(lines, "")
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(lines, "\n"), footer)
}

// detailView lays out the task header, the glamour-rendered markdown in its
// viewport, and a scroll-position footer.
func (m Model) detailView() string {
	w, _ := m.dims()
	r := m.detailRow
	crumb := crumbStyle.Render(m.breadcrumb()+" › "+r.ID) + "  " +
		statusStyle(r.Status).Render(r.Status) + "  " +
		dimStyle.Render(gatesCell(r))
	header := m.headerLine(crumb, w)
	footer := " " + dimStyle.Render(fmt.Sprintf("%3.0f%%", m.detailVP.ScrollPercent()*100))
	return lipgloss.JoinVertical(lipgloss.Left, header, m.detailVP.View(), footer)
}

// headerLine wraps content in the full-width rounded header box.
func (m Model) headerLine(content string, w int) string {
	return headerBox.Width(max(w-2, 1)).Render(ansi.Truncate(content, max(w-4, 1), "…"))
}

// footerLine is the subtle status line: subtree counts and drift total, or
// the last scan error.
func (m Model) footerLine(w int) string {
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

// bodyLines renders the board's lines — sub-boards, then each section header
// and its rows — and returns the index of the selected line.
func (m Model) bodyLines(w int) (lines []string, sel int) {
	b, ok := m.board()
	if !ok {
		return []string{" " + dimStyle.Render("no boards found")}, 0
	}
	cursor := m.cursor()
	idx := 0
	nameW := maxSubNameWidth(b.Subs)
	for i := range b.Subs {
		if idx == cursor {
			sel = len(lines)
		}
		lines = append(lines, subLine(b.Subs[i], nameW, idx == cursor, w))
		idx++
	}
	idW := maxIDWidth(b.Sections)
	driftW := maxDriftWidth(b.Sections)
	for _, s := range b.Sections {
		lines = append(lines, "", " "+sectionStyle.Render(s.Name))
		for ri := range s.Rows {
			if idx == cursor {
				sel = len(lines)
			}
			lines = append(lines, rowLine(s.Rows[ri], idW, driftW, idx == cursor, w))
			idx++
		}
	}
	if idx == 0 {
		lines = append(lines, " "+dimStyle.Render("no open tasks"))
	}
	return lines, sel
}

// subLine renders one sub-board: name, title, live counts.
func subLine(s Sub, nameW int, selected bool, w int) string {
	title := s.Title
	if selected {
		title = selStyle.Render(title)
	}
	line := cursorMark(selected) +
		accentStyle.Render(pad(s.Name, nameW)) + "  " +
		title + "  " +
		dimStyle.Render(fmt.Sprintf("%d/%d done", s.Done, s.Total))
	return ansi.Truncate(line, w, "…")
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

// window slices lines to visible rows, keeping the selected line roughly
// centered.
func window(lines []string, sel, visible int) []string {
	if len(lines) <= visible {
		return lines
	}
	top := clamp(sel-visible/2, 0, len(lines)-visible)
	return lines[top : top+visible]
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
