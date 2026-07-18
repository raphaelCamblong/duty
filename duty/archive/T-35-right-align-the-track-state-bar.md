---
id: T-35
title: Right-align the track state bar
status: done
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
- [x] Frames at 120×35 and 70×20: bars flush right, same start column across track rows — recorded in the report.
- [x] No file outside internal/tui (+ spec §8) modified; TestStartupPerformance green.
- [x] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok.

## Report

Track rows now mirror `taskLine`'s right-column technique: `name/  Title` left,
the title padded to `max(w-fixed, minTitleWidth)`, then a right-aligned column of
`trackBarWidth(14) + 2 + countW` cells — the state bar and a right-aligned dim
total count flush at the line end. `countW` (widest total count across the board's
tracks) is measured once in `newDelegate`, so the bar column starts at the same x
on every track row regardless of title length or count width. Narrow widths
ellipsis-truncate the title first; the bar is never dropped (dim `empty` still
fills the reserved column for an empty subtree).

Frames from `TestTrackBarRightAligned` — two sibling tracks with mismatched title
lengths and count widths (`api/` title "API", 2 tasks; `frontend/` title "The
frontend web application", 11 tasks). Bars flush right, same start column on both
rows; counts (`2`, `11`) right-aligned to a common edge.

120×35 (bar start col 100):
```
│ ❯ api/       API                                                                                  ██████████████   2 │
│   frontend/  The frontend web application                                                         ██████████████  11 │
```

70×20 (bar start col 50):
```
│ ❯ api/       API                                ██████████████   2 │
│   frontend/  The frontend web application       ██████████████  11 │
```

`TestStartupPerformance` green (best of 5 ~1.8ms). Full suite green, golangci-lint
0 issues, gofumpt clean, go vet clean. Changes confined to `internal/tui`
(`entry.go`, `view.go`) plus the spec §8 track-row sentence and the new test.

Dogfood: verified via the fresh binary at 120x35 and 70x20 — track bars flush right, same start column across tracks. Full report with frames above.

Simplify-pass addendum: extracted trackRightWidth(countW) beside trackBarCell (internal/tui/view.go) as the single owner of the trailing state-column width (trackBarWidth+2+countW). trackLine's rightW (internal/tui/entry.go) now calls it instead of re-deriving the composition, so the title-pad and the cell can no longer desync — one source of truth naming the constraint the two sites previously only implied. Behavior-preserving: full suite green, golangci-lint 0 issues, gofumpt clean, go vet clean, build ok. The reviewers' other observations (countW/maxSubCountWidth mirroring the existing maxSub* pattern, the irreducible empty-vs-count branch, taskLine's parallel inline arithmetic) were left as-is — clean, not defects.
