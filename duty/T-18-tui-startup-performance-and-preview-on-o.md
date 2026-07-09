---
id: T-18
title: TUI startup performance and preview on open
status: todo
blocked-by: []
---

# T-18 — TUI startup performance and preview on open

## Goal
`duty tui` opens instantly, and the right panel exists only when a task is opened —
browsing is a full-width list under the header.

## Read first
`internal/tui/model.go`, `view.go`, `run.go`; how glamour renderers and
termenv/lipgloss background detection behave (both are known startup costs:
renderer construction loads styles; auto background detection blocks on an OSC
terminal query).

## Scope
- **Measure first.** Add a benchmark/timed test around model construction + first
  `View()` on a realistic fixture tree (≈20 tasks, 3 tracks). Record the number
  before any fix, in the report.
- **Kill the known hotspots:**
  - ONE glamour renderer for the whole program, built lazily on the FIRST task
    open, reused ever after; rebuilt only on width change while a preview is open.
  - Terminal background detection runs exactly once, in `run.go` before the
    program starts (only when theme = `auto`); the result is passed down — no
    style/renderer construction may trigger another query mid-frame.
  - Zero glamour work while browsing: no render on selection change, no cache
    warming. Markdown renders only on open.
- **UX change:** the right panel is GONE while browsing (left list takes the full
  width, header + footer unchanged — top and left stay exactly as they are).
  `enter`/double-click on a task opens the split: list stays left, rendered task
  right, focus on the preview; `esc` closes back to full-width browsing. `enter`
  on a track still descends. Track summary card: shown in the same on-open way
  when `enter` is pressed on a track with the preview already open — otherwise
  tracks never open a panel (they descend). The `e` editor binding stays (it
  costs nothing at runtime).
- Re-measure after; record before/after in the report. Update spec §8's layout
  paragraph (preview appears on open) in the same change.

## Out of scope
Removing the editor binding; watcher/scan changes beyond what measurement proves
necessary; any CLI change; layout changes to header/left panel.

## Gates
- [ ] Timed test: model construction + first `View()` on the fixture completes in
  under 100ms (no terminal queries, no glamour construction on that path) —
  before/after numbers in the report.
- [ ] Update tests: no right panel in the browsing frame; `enter` on a task shows
  it; `esc` removes it; `enter` on a track descends. Headless 120×35 frames for
  both states recorded in the report.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report
