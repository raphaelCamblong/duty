package tests

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"

	"github.com/raphaelCamblong/duty/internal/tui"
)

// miniDot is the spinner's first frame — the glyph a freshly built model shows
// beside an in-progress row before any tick advances it.
var miniDot = spinner.MiniDot.Frames[0]

func TestSpinnerGlyphOnInProgressRow(t *testing.T) {
	root := tuiTree(t) // T-01 in-progress, T-02 todo
	m := newTUIModelSize(t, root, 120, 35)

	t.Run("the in-progress row carries the glyph beside its status", func(t *testing.T) {
		line := frameLine(t, m.View().Content, "T-01")
		if !strings.Contains(line, "in-progress") || !strings.Contains(line, miniDot) {
			t.Errorf("in-progress row missing the spinner glyph %q: %q", miniDot, line)
		}
	})

	t.Run("a todo row shows no glyph", func(t *testing.T) {
		line := frameLine(t, m.View().Content, "T-02")
		if strings.Contains(line, miniDot) {
			t.Errorf("todo row unexpectedly shows the spinner glyph: %q", line)
		}
	})

	t.Run("the preview header carries the glyph when the open task is in-progress", func(t *testing.T) {
		// A narrow terminal takes the preview full-screen, so the list (and its
		// own glyph) is gone — the only glyph left is the header's.
		pm := newTUIModelSize(t, root, 70, 20)
		pm, _ = press(t, pm, "j")     // backend/ -> T-01
		pm, _ = press(t, pm, "enter") // open T-01 full-screen
		if pm.DetailID() != "T-01" {
			t.Fatalf("preview did not open on T-01: DetailID=%q", pm.DetailID())
		}
		if frame := pm.View().Content; !strings.Contains(frame, miniDot) {
			t.Errorf("in-progress preview header missing the spinner glyph:\n%s", frame)
		}
	})
}

// TestSpinnerTickLifecycle is the heart of T-56: ticks run only while the
// snapshot holds an in-progress task. A quiet board arms nothing; gaining one on
// a re-scan arms the loop; losing the last one stops it — the trailing tick
// schedules nothing further.
func TestSpinnerTickLifecycle(t *testing.T) {
	root := initDuty(t)
	mustDuty(t, root, "create", "task", "Alpha") // T-01 todo
	mustDuty(t, root, "create", "task", "Beta")  // T-02 todo
	m := newTUIModelSize(t, root, 120, 35)

	if m.Spinning() {
		t.Fatal("spinner armed on a board with no in-progress task")
	}

	// Gaining an in-progress task on a re-scan arms the loop.
	mustDuty(t, root, "status", "T-01", "in-progress")
	m = rescan(t, m)
	if !m.Spinning() {
		t.Fatal("spinner did not arm when the snapshot gained an in-progress task")
	}

	// While in-progress work remains, a tick advances the frame and reschedules.
	nm, tick := m.Update(spinner.TickMsg{})
	m = nm.(tui.Model)
	if tick == nil {
		t.Error("spinner stopped ticking while an in-progress task remains")
	}

	// Completing the last in-progress task stops the loop: the trailing tick
	// schedules nothing and the loop reads as stopped.
	mustDuty(t, root, "status", "T-01", "done")
	m = rescan(t, m)
	nm, tick = m.Update(spinner.TickMsg{})
	m = nm.(tui.Model)
	if tick != nil {
		t.Error("spinner scheduled another tick after the last in-progress task completed")
	}
	if m.Spinning() {
		t.Error("spinner still marked live after the last in-progress task completed")
	}
}

// rescan drives a manual re-scan ("r") and applies the resulting snapshot,
// the exact path the watcher takes when files change on disk.
func rescan(t *testing.T, m tui.Model) tui.Model {
	t.Helper()
	_, cmd := press(t, m, "r")
	if cmd == nil {
		t.Fatal("r returned no re-scan command")
	}
	nm, _ := m.Update(cmd())
	return nm.(tui.Model)
}
