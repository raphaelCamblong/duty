---
id: T-11
title: TUI mouse, motion, charts polish
status: todo
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
- [ ] Update-transition tests for mouse messages: click → selection, double-click →
  detail, wheel → scroll target.
- [ ] Observed manually and noted in the report: mouse selection/open/scroll work,
  motion is smooth, header bar reflects the real status mix, `?` toggles hints.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
