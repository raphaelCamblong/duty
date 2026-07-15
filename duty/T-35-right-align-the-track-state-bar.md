---
id: T-35
title: Right-align the track state bar
status: todo
blocked-by: []
---

# T-35 — Right-align the track state bar

## Goal
Track rows read like task rows: name and title on the left, the state bar and
its count right-aligned at the end of the line — not inlined after the title.

## Read first
`internal/tui/entry.go` (`trackLine` — current layout: name, title, bar, count
inline) and how `taskLine` right-aligns its status/gates/age columns;
`task-system-spec.md` §8 track-row sentence.

## Scope
- Track row layout becomes: `name/  Title` left, flexible gap, then the
  14-cell state bar + dim total count right-aligned at the line's end — the
  bar column starts at the same x on every track row (mirror `taskLine`'s
  right-column technique).
- Narrow terminals: the title ellipsis-truncates first (existing behavior);
  the bar+count column is reserved like task rows' right columns; below the
  width where task rows hide optional columns, keep the bar (it IS the state)
  and drop nothing else new.
- Display-only, `internal/tui` only; spec §8 track-row sentence updated in the
  same change.
- Tests: entry/frame checks at 120×35 and 70×20 showing the bar flush right
  and aligned across multiple tracks; `TestStartupPerformance` green.

## Out of scope
Bar content/colors/width (T-24 owns those); task rows; CLI output; anything
outside `internal/tui` except the spec sentence.

## Gates
- [ ] Frames at 120×35 and 70×20: bars flush right, same start column across track rows — recorded in the report.
- [ ] No file outside internal/tui (+ spec §8) modified; TestStartupPerformance green.
- [ ] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok.

## Report
