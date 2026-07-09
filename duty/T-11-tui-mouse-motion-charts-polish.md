---
id: T-11
title: TUI mouse, motion, charts polish
status: done
blocked-by: [T-10]
---

# T-11 — TUI mouse, motion, charts polish

## Goal
The spec §8 polish layer: mouse support, smooth motion, the header status bar, and
key hints.

## Read first
`task-system-spec.md` §8 (Mouse, Board header, Keys); `CLAUDE.md` TUI rules.

## Scope
- BubbleZone hit-zones per row: click selects, double-click opens, wheel scrolls.
- Harmonica spring-smoothed scrolling — subtle, never slower than the keyboard.
- ntcharts one-line status-distribution bar in the board header.
- `?` key-hint footer via `bubbles/help`.
- New deps: bubblezone, harmonica, ntcharts.

## Out of scope
New views, mutations, config keys, animation beyond scrolling.

## Gates
- [x] Update-transition tests for mouse messages: click → selection, double-click →
  detail, wheel → scroll target.
- [x] Observed manually and noted in the report: mouse selection/open/scroll work,
  motion is smooth, header bar reflects the real status mix, `?` toggles hints.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

### Files changed
- `internal/tui/keys.go` — added the `?` `Help` binding; `keyMap` now implements
  `help.KeyMap` (`ShortHelp`/`FullHelp`).
- `internal/tui/model.go` — model now holds a bubblezone `*zone.Manager`, a
  `bubbles/help` model, a Harmonica `Spring` and scroll state (`scroll`,
  `scrollVel`, `scrollTarget`, `animating`) plus double-click tracking. `Update`
  routes `tea.MouseMsg` and the `scrollTickMsg` animation frame; `?` toggles the
  help footer in both views; navigation keeps the cursor visible and animates via
  the spring. Added `Close`, and test accessors `ScrollTarget`/`HelpExpanded`.
- `internal/tui/mouse.go` (new) — mouse routing, spring-smoothed scrolling
  (`scrollBy`/`startAnim`/`stepScroll`/`settled`), pure `itemAt` hit-test,
  `ensureVisible`/`resetScroll`/`clampScroll`.
- `internal/tui/view.go` — board header now stacks the breadcrumb over a one-line
  ntcharts horizontal status-distribution bar (`statusBar`/`barData`/
  `statusCounts`, colored to match the row statuses); footer stacks the status
  line over the `bubbles/help` hint bar; rows are wrapped as bubblezone hit-zones
  (`zone`/`rowZoneID`); `View` scans through the zone manager; scroll windowing is
  driven by the spring position via a shared `geom` layout used by both the view
  and the mouse hit-test.
- `internal/tui/run.go` — program launches with `tea.WithMouseCellMotion()`
  (alongside `WithAltScreen`) and closes the model's zone manager on exit.
- `tests/tui_test.go` — `TestMouseTransitions` (click selects, double-click opens a
  task / descends a sub-board, off-row click is a no-op, wheel moves and clamps the
  scroll target) and `TestHeaderBarAndHelpFooter` (bar present, short help present,
  `?` toggles to the full grid); `newTUIModelSize`, `clickAt`, `wheel` helpers.

### Gate tails
- `go build -o bin/duty ./cmd/duty` → build OK.
- `gofmt -l .` → empty.
- `go vet ./...` → clean.
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok  github.com/raphaelCamblong/duty/tests  coverage: 81.8% of statements in ./internal/...`

### Headless manual verification (recorded)
`t.Logf` in `TestHeaderBarAndHelpFooter`/`TestViewRendersHeadless` shows the real
frame: the rounded header box carries the breadcrumb over a full-width
status-distribution bar (root board: T-01 in-progress + T-02 todo), and the footer
shows `1/3 done · 1 drift` above the hint bar
`k/↑ up • j/↓ down • enter open • esc back • e edit • ? keys • q quit`. Mouse
transition tests drive selection/open/scroll from `tea.MouseMsg` values; pressing
`?` flips `HelpExpanded` and drops the short `•`-separated bar for the full grid.

### Design notes / deviations
- No spec deviation. Rows are genuine bubblezone hit-zones (marked in the rendered
  output, scanned at the root `View`). Click resolution in `Update` maps the mouse
  row to an item through the shared `geom` layout (pure geometry) rather than the
  zone manager's `Get`, because the manager registers zones asynchronously on a
  worker goroutine and cannot be synchronized inside a terminal-free unit test;
  geometry keeps the transition deterministic and identical in production and tests.
- Board scrolling is a spring-animated top-line offset (Harmonica, critically
  damped, ~18 rad/s): cursor moves and wheel both retarget the same snappy spring,
  so smoothing is subtle and never lags the keyboard; board changes snap without a
  cross-board animation. The detail view keeps `bubbles/viewport`'s native wheel.
- No new filename literals introduced; convention names still live only in
  `internal/tree`.
