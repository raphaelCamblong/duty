package tui

import (
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
)

type scrollTickMsg struct{}

const (
	scrollFPS   = 60
	scrollFreq  = 18.0
	wheelStep   = 3
	doubleClick = 400 * time.Millisecond
)

func scrollTickCmd() tea.Cmd {
	return tea.Tick(time.Second/scrollFPS, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

// handleMouse routes a mouse event: the wheel scrolls the hovered panel and
// a left press selects the entry (left) or focuses the preview (right), a
// second press on the same entry opening it. Bubble Tea v2 encodes the action
// in the message type, so the routing switches on that rather than a field.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch e := msg.(type) {
	case tea.MouseWheelMsg:
		switch e.Button {
		case tea.MouseWheelUp:
			return m.wheel(msg, -1)
		case tea.MouseWheelDown:
			return m.wheel(msg, 1)
		}
	case tea.MouseClickMsg:
		if e.Button == tea.MouseLeft {
			return m.click(msg)
		}
	}
	return m, nil
}

func (m Model) wheel(msg tea.MouseMsg, dir int) (tea.Model, tea.Cmd) {
	if m.overPreview(msg) {
		m.scrollTarget = clamp(m.scrollTarget+dir*wheelStep, 0, m.previewMax())
		return m.startAnim()
	}
	return m.moveSelection(dir), nil
}

// overPreview reports whether a mouse event targets the open preview: inside
// its zone when the split shows, anywhere while it is the single panel; never
// while browsing.
func (m Model) overPreview(msg tea.MouseMsg) bool {
	if !m.previewOpen {
		return false
	}
	if !m.wide() {
		return true
	}
	return m.zones.Get(zonePreview).InBounds(msg)
}

// click routes a left press: a breadcrumb segment jumps to that ancestor
// track, the open preview takes focus, otherwise the entry under the pointer
// is selected (a second press on it opens it).
func (m Model) click(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	for _, p := range m.crumbChain() {
		if m.zones.Get(crumbZone(p)).InBounds(msg) {
			return m.enterBoard(p), nil
		}
	}
	if m.split() && m.zones.Get(zonePreview).InBounds(msg) {
		m.focus = focusPreview
		return m, nil
	}
	for i := range m.list.VisibleItems() {
		if m.zones.Get(itemZone(i)).InBounds(msg) {
			return m.clickItem(i), nil
		}
	}
	return m, nil
}

func (m Model) clickItem(index int) Model {
	m.focus = focusList
	m.list.Select(index)
	now := time.Now()
	if m.lastClick == index && now.Sub(m.lastClickAt) < doubleClick {
		m.lastClick = -1
		return m.open()
	}
	m.lastClick, m.lastClickAt = index, now
	return m
}

func (m Model) previewMax() int {
	return max(m.preview.TotalLineCount()-m.preview.Height(), 0)
}

func (m Model) settleAt(offset int) Model {
	m.scroll, m.scrollVel = float64(offset), 0
	m.scrollTarget, m.animating = offset, false
	return m
}

func (m Model) startAnim() (tea.Model, tea.Cmd) {
	if m.animating || m.settled() {
		return m, nil
	}
	m.animating = true
	return m, scrollTickCmd()
}

func (m Model) stepScroll() (tea.Model, tea.Cmd) {
	m.scroll, m.scrollVel = m.spring.Update(m.scroll, m.scrollVel, float64(m.scrollTarget))
	if m.settled() {
		m = m.settleAt(m.scrollTarget)
		m.preview.SetYOffset(m.scrollTarget)
		return m, nil
	}
	m.preview.SetYOffset(int(math.Round(m.scroll)))
	return m, scrollTickCmd()
}

func (m Model) settled() bool {
	return math.Abs(m.scroll-float64(m.scrollTarget)) < 0.2 && math.Abs(m.scrollVel) < 0.2
}
