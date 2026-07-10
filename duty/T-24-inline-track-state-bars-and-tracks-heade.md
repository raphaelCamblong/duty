---
id: T-24
title: Inline track state bars and tracks header
status: todo
blocked-by: []
---

# T-24 — Inline track state bars and tracks header

## Goal
Track rows read at a glance: a small inline status-distribution bar (the header
bar, miniaturized) instead of textual counts, under their own "Tracks" section
header.

## Read first
`internal/tui/entry.go` (compactDelegate, section headers), `view.go` (the header
ntcharts bar and `statusRollup`), `task-system-spec.md` §8.

## Scope
- **"Tracks" section header** above the sub-track rows in the list, styled and
  behaving exactly like the existing task section headers ("Open tasks"):
  non-selectable, skipped by navigation, hidden while filtering.
- **Inline state bar per track row**: `name  title  ▰▰▰▰▰▰▱▱  n` — a fixed-width
  (~14 cells) horizontal bar of colored segments proportional to the track's
  subtree per-status counts, using the exact status colors of the header bar,
  followed by a dim total-task count. Rounding rule: every non-zero status gets
  at least one cell. Reuse the header bar's rendering at row scale if it
  composes cleanly at that width; otherwise colored block runs via lipgloss —
  same palette either way. Empty track: dim `empty` (unchanged).
- The textual per-status rollup moves out of the row but STAYS in the track
  preview card and the track-selected preview (numbers on open, bar while
  browsing).
- Spec §8's track-row sentence updated in the same change.
- Do NOT regress T-18: no glamour work or terminal queries while browsing;
  `TestStartupPerformance` stays green.

## Out of scope
Header bar, task rows, preview card layout, CLI output (`get tracks` keeps its
textual counts).

## Gates
- [ ] Update/view tests: track row renders the bar segments proportionally
  (fixture with mixed statuses), non-zero statuses always visible, "Tracks"
  header present, skipped by j/k navigation and absent while filtering.
- [ ] Headless frames at 120×35 and 70×20 recorded in the report; nothing ragged.
- [ ] `TestStartupPerformance` green; full suite green
  (`go test ./tests/... -coverpkg=./internal/... -count=1`); `gofmt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report
