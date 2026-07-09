package tui

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// scrollTickMsg advances the preview scroll spring one animation frame.
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

// handleMouse routes a mouse event: the wheel scrolls the hovered panel and
// a left press selects the entry (left) or focuses the preview (right), a
// second press on the same entry opening it.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.wheel(msg, -1)
	case tea.MouseButtonWheelDown:
		return m.wheel(msg, 1)
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.click(msg)
	}
	return m, nil
}

// wheel scrolls whichever panel the pointer hovers: the preview glides on
// its spring, the list moves its selection.
func (m Model) wheel(msg tea.MouseMsg, dir int) (tea.Model, tea.Cmd) {
	if m.overPreview(msg) {
		m.scrollTarget = clamp(m.scrollTarget+dir*wheelStep, 0, m.previewMax())
		return m.startAnim()
	}
	return m.moveSelection(dir), nil
}

// overPreview reports whether a mouse event targets the preview: inside its
// zone when both panels show, anywhere while it is the single panel.
func (m Model) overPreview(msg tea.MouseMsg) bool {
	if !m.wide() {
		return m.focus == focusPreview
	}
	return m.zones.Get(zonePreview).InBounds(msg)
}

// click selects the entry under the pointer or focuses the clicked preview;
// a second press on the same entry within doubleClick opens it.
func (m Model) click(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.wide() && m.zones.Get(zonePreview).InBounds(msg) {
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

// clickItem selects the visible entry at index; a double click opens it.
func (m Model) clickItem(index int) Model {
	m.focus = focusList
	m.list.Select(index)
	m = m.syncPreview(false)
	now := time.Now()
	if m.lastClick == index && now.Sub(m.lastClickAt) < doubleClick {
		m.lastClick = -1
		return m.open()
	}
	m.lastClick, m.lastClickAt = index, now
	return m
}

// previewMax is the preview's highest top line.
func (m Model) previewMax() int {
	return max(m.preview.TotalLineCount()-m.preview.Height, 0)
}

// settleAt pins the preview spring at offset, stopping any animation.
func (m Model) settleAt(offset int) Model {
	m.scroll, m.scrollVel = float64(offset), 0
	m.scrollTarget, m.animating = offset, false
	return m
}

// startAnim begins the spring animation unless it is already running or the
// preview is already at rest at its target.
func (m Model) startAnim() (tea.Model, tea.Cmd) {
	if m.animating || m.settled() {
		return m, nil
	}
	m.animating = true
	return m, scrollTickCmd()
}

// stepScroll advances the spring toward the target, moving the preview and
// snapping once the motion has settled.
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

// settled reports whether the spring has reached its target to within a
// fraction of a line.
func (m Model) settled() bool {
	return math.Abs(m.scroll-float64(m.scrollTarget)) < 0.2 && math.Abs(m.scrollVel) < 0.2
}
