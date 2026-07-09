---
id: T-10
title: TUI live board viewer
status: todo
blocked-by: [T-05, T-08]
---

# T-10 — TUI live board viewer

## Goal
`duty tui` per spec §8: navigable board + detail views, keyboard driven, refreshing
live from the filesystem.

## Read first
`task-system-spec.md` §8, §7 (theme/editor); `CLAUDE.md` TUI rules (pure update
logic, bubbles before hand-rolling, lipgloss only).

## Scope
- `internal/tui`: a pure scan → view-model layer (boards, section/row order from each
  `BOARD.md`, truth + gate counts from files, drift badges), then the Bubble Tea
  model/update/view on top.
- Board view: breadcrumb (H1 titles), sub-board rows with live counts
  (`Backend  3/7 done`), sections, rows with colored status (todo dim, in-progress
  yellow, blocked red, done green), gate progress, `⚠` drift badge.
- Detail view: task markdown via glamour inside `bubbles/viewport`.
- Keys: `j/k`, `enter` (descend / open), `esc` (up / close), `e` → `$EDITOR` from
  config (suspend/resume), `q`.
- Live refresh: fsnotify watch per directory, ~100 ms debounce, full re-scan on any
  event, re-walk to add watches when directories appear.
- Theme from config (`auto|dark|light`). Read-only: zero writes from the TUI.
- New deps: bubbletea, bubbles, lipgloss, glamour, fsnotify.

## Out of scope
Mouse, harmonica, ntcharts, `?` help footer (all T-11); any mutation path.

## Gates
- [ ] View-model tests on a fixture tree in `tests/`: ordering matches the boards,
  counts and drift computed from files, archived ignored.
- [ ] Update-transition tests: key messages produce the expected navigation states
  (no terminal needed).
- [ ] Observed manually and noted in the report: `duty tui` renders the real tree,
  and an external `duty status` + an `$EDITOR` save each appear without a keypress.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
