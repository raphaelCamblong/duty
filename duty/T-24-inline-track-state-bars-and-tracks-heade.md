---
id: T-24
title: Inline track state bars and tracks header
status: done
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
- [x] Update/view tests: track row renders the bar segments proportionally
  (fixture with mixed statuses), non-zero statuses always visible, "Tracks"
  header present, skipped by j/k navigation and absent while filtering.
- [x] Headless frames at 120×35 and 70×20 recorded in the report; nothing ragged.
- [x] `TestStartupPerformance` green; full suite green
  (`go test ./tests/... -coverpkg=./internal/... -count=1`); `gofmt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report


### T-24 — Inline track state bars and tracks header

**Files changed**
- internal/tui/entry.go — "Tracks" section header above sub-track rows; trackLine now renders the inline bar cell instead of the textual rollup.
- internal/tui/view.go — BarCells (proportional allocation, min-1 per non-zero status, largest-remainder fill), trackBar, trackBarCell, totalCount; statusBar deduped onto totalCount; trackBarWidth=14.
- internal/tui/model.go — fixSelection now clamps an out-of-range cursor back onto the visible list (filtering can strand it once a header shifts the initial index).
- task-system-spec.md — §8 left-panel track-row sentence updated (Tracks header + inline bar + dim total, rollup moved to preview card).
- tests/tui_test.go — TestTrackBarCells (pure allocation) + TestTracksHeaderAndInlineBar (header present, bar segments, nav-skip, filter-absence).

**Gate tails**
- gofmt -l internal/ tests/ : clean
- go vet ./... : clean
- go build -o bin/duty ./cmd/duty : ok
- go test ./tests/... -coverpkg=./internal/... -count=1 : ok, coverage 85.3%
- TestStartupPerformance : green, best of 5 = 2.48ms (< 100ms; no glamour/terminal query on the browse path)

**Headless frame 120x35 (browse)**
```
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ Board                                                                                                                │
│ 1 in-progress · 1 todo · 1 blocked · 1 done  █████████████████▊█████████████████▌█████████████████▎█████████████████ │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│  Tracks                                                                                                              │
│ ❯ backend/  Backend services  ██████████████  2                                                                      │
│  Open tasks                                                                                                          │
│   T-01  Alpha task                                                                                in-progress        │
│   T-02  Beta task                                                                                 todo               │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 k/↑ up • j/↓ down • ↵ open • esc back • tab panel • / filter • e edit • r refresh • ? keys • q quit
```

**Headless frame 70x20 (browse)**
```
╭────────────────────────────────────────────────────────────────────╮
│ Board                                                              │
│ 1 in-progress · 1 todo · 1 blocked · 1 done  █████▎████▌████▊█████ │
╰────────────────────────────────────────────────────────────────────╯
╭────────────────────────────────────────────────────────────────────╮
│  Tracks                                                            │
│ ❯ backend/  Backend services  ██████████████  2                    │
│  Open tasks                                                        │
│   T-01  Alpha task                              in-progress        │
│   T-02  Beta task                               todo               │
╰────────────────────────────────────────────────────────────────────╯
 k/↑ up • j/↓ down • ↵ open • esc back • tab panel • / filter • e edi…
```
Both frames non-ragged (asserted by TestFrameAudit across 120x35..60x16).

**Deviations**
- Row bar drawn as lipgloss colored "█" runs (not the ntcharts header renderer): ntcharts floor-rounding drops small segments, which conflicts with the "every non-zero status ≥ 1 cell" rule — the task's stated fallback. Same palette (statusColor) either way.
- BarCells was exported so the proportionality / min-1 rounding is unit-tested directly: lipgloss strips color in the headless test profile, so per-status segments are indistinguishable in a rendered frame.
- Bonus fix in fixSelection (out-of-range cursor clamp) needed because the new "Tracks" header shifts the initial selection index, exposing a latent bubbles quirk (FilterMatchesMsg does not re-clamp the cursor).
