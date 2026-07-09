package tui

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// scrollTickMsg advances the scroll spring one animation frame.
type scrollTickMsg struct{}

const (
	scrollFPS   = 60
	scrollFreq  = 18.0
	wheelStep   = 3
	doubleClick = 400 * time.Millisecond
)

// scrollTickCmd schedules the next animation frame.
func scrollTickCmd() tea.Cmd {
	return tea.Tick(time.Second/scrollFPS, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

// handleMouse routes a mouse event: the detail viewport scrolls itself; on the
// board the wheel spring-scrolls and a left press selects the row under the
// pointer, a second press on the same row opening it.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.detailOpen {
		var cmd tea.Cmd
		m.detailVP, cmd = m.detailVP.Update(msg)
		return m, cmd
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.scrollBy(-wheelStep).startAnim()
	case tea.MouseButtonWheelDown:
		return m.scrollBy(wheelStep).startAnim()
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.click(msg)
	}
	return m, nil
}

// scrollBy shifts the scroll target by delta lines, clamped to the content.
func (m Model) scrollBy(delta int) Model {
	m.scrollTarget = clamp(m.scrollTarget+delta, 0, m.geom().maxTop)
	return m
}

// click selects the row under the pointer; a second press on the same row
// within doubleClick opens it (descend or detail).
func (m Model) click(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	idx, ok := m.itemAt(msg.Y)
	if !ok {
		return m, nil
	}
	m.cursors[m.path] = idx
	now := time.Now()
	if m.lastClick == idx && now.Sub(m.lastClickAt) < doubleClick {
		m.lastClick = -1
		return m.open().startAnim()
	}
	m.lastClick, m.lastClickAt = idx, now
	return m.ensureVisible().startAnim()
}

// itemAt maps a screen row to a selectable item index using the same layout
// the view renders, returning false for the header, footer, blanks and
// section labels.
func (m Model) itemAt(y int) (int, bool) {
	g := m.geom()
	bodyY := y - g.headerH
	if bodyY < 0 || bodyY >= g.visible {
		return 0, false
	}
	li := g.top + bodyY
	if li < 0 || li >= len(g.lineItem) {
		return 0, false
	}
	if it := g.lineItem[li]; it >= 0 {
		return it, true
	}
	return 0, false
}

// ensureVisible nudges the scroll target so the cursor's line stays within the
// visible window.
func (m Model) ensureVisible() Model {
	g := m.geom()
	t := m.scrollTarget
	if g.selLine < t {
		t = g.selLine
	} else if g.selLine >= t+g.visible {
		t = g.selLine - g.visible + 1
	}
	m.scrollTarget = clamp(t, 0, g.maxTop)
	return m
}

// resetScroll snaps to the top on a board change, then keeps the remembered
// cursor visible without animating across boards.
func (m Model) resetScroll() Model {
	m.scrollTarget = 0
	m = m.ensureVisible()
	m.scroll, m.scrollVel, m.animating = float64(m.scrollTarget), 0, false
	return m
}

// clampScroll re-bounds the scroll to the content after a resize or re-scan.
func (m Model) clampScroll() Model {
	g := m.geom()
	m.scrollTarget = clamp(m.scrollTarget, 0, g.maxTop)
	m.scroll = math.Min(math.Max(m.scroll, 0), float64(g.maxTop))
	return m
}

// startAnim begins the spring animation unless it is already running or the
// view is already at rest at its target.
func (m Model) startAnim() (tea.Model, tea.Cmd) {
	if m.animating || m.settled() {
		return m, nil
	}
	m.animating = true
	return m, scrollTickCmd()
}

// stepScroll advances the spring toward the target, snapping and stopping once
// the motion has settled.
func (m Model) stepScroll() (tea.Model, tea.Cmd) {
	m.scroll, m.scrollVel = m.spring.Update(m.scroll, m.scrollVel, float64(m.scrollTarget))
	if m.settled() {
		m.scroll, m.scrollVel, m.animating = float64(m.scrollTarget), 0, false
		return m, nil
	}
	return m, scrollTickCmd()
}

// settled reports whether the spring has reached its target to within a
// fraction of a line.
func (m Model) settled() bool {
	return math.Abs(m.scroll-float64(m.scrollTarget)) < 0.2 && math.Abs(m.scrollVel) < 0.2
}
