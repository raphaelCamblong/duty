---
id: T-60
title: Migrate to Bubble Tea v2
status: done
blocked-by: []
---

# T-60 — Migrate to Bubble Tea v2

## Goal
The TUI runs on Bubble Tea v2 — which removes v1's package-init terminal
query (tea_init.go:21), the last place duty's process interrogates a
terminal it doesn't own.

## Read first
T-59's report (the goroutine-1 stack proving the init query; why it's
unfixable in-process on v1), bubbletea v2 migration guide + release notes,
the paired bubbles/lipgloss versions v2 expects.

## Scope
- Bump bubbletea (+ bubbles/lipgloss as v2 requires) and migrate
  internal/tui to the v2 API: program options, Init/Update/Cmd changes,
  key/mouse message shapes, whatever the guide lists.
- Behavior frozen: every existing TUI test (frames, transitions, startup
  perf, spinner lifecycle, archive toggle, filter guard) passes with only
  mechanical signature updates — no visual or interaction change.
- Verify under a mute pty: startup never blocks on terminal queries, early
  keystrokes are never eaten (the T-59 rig, now with a fair oracle).
- Docs untouched unless behavior shifts (it must not).

## Out of scope
New v2-only features; theming changes; CLI.

## Gates
- [x] No terminal query anywhere in the process (mute-pty startup instant,
  first frame < 1s, piped early keys all delivered).
- [x] Full suite green with mechanical-only test edits (listed); startup
  perf and frame audits unchanged.
- [x] just check green; go.mod tidy stable.

## Report

### 2026-07-18 15:42 — done

Migrated the TUI to Bubble Tea v2 (charm.land/bubbletea/v2 v2.0.8) with the paired
charm.land lipgloss/v2, bubbles/v2, glamour/v2, huh/v2 and github.com bubblezone/v2,
ntcharts/v2. huh moved to v2 too so bubbletea v1 leaves the binary entirely — its
package-init OSC background query (the T-59 freeze) is gone from the process.

API deltas: View() returns tea.View (AltScreen + MouseModeCellMotion set on it,
replacing the removed program options); KeyMsg/MouseMsg are interfaces (KeyPressMsg,
MouseClickMsg/MouseWheelMsg); viewport/help getters are methods; lipgloss v2 dropped
AdaptiveColor/SetHasDarkBackground so the theme resolves light/dark from config mode.
lipgloss v2 Width/Height now include border+padding, so bordered boxes are sized by
GetHorizontalFrameSize/GetVerticalFrameSize to keep identical dimensions.

Behavior frozen: full suite green, golden frames byte-identical once ANSI-stripped.
Verified under a mute pty: first frame at ~62ms, an early keystroke sent before render
is delivered (selection moved) — the fair oracle T-59 lacked.
