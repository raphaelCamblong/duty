---
id: T-39
title: Palette-true light theme via status chips
status: todo
blocked-by: []
---

# T-39 — Palette-true light theme via status chips

## Goal
Light terminals show the palette EXACTLY as given: status words become chips —
palette hue as background, indigo ink on top — and bars use the raw hues.
Dark theme stays as T-38 shipped it (approved).

## Read first
`internal/tui/view.go` palette block + `statusColor`/`statusStyle` (T-38 +
its simplify addendum); why fg-only fails on light: cream 1.2:1, peach 1.9,
olive 2.3 on white — only indigo (17.8) and marginally bronze (3.0) read.

## Scope
- Dark theme: UNCHANGED (user-approved).
- Light theme, statuses as chips: `statusStyle` becomes theme-aware — dark →
  Foreground(statusColor) as today; light → Background(raw palette hue) +
  indigo `#1e0f37` foreground, one space of horizontal padding: done=olive bg,
  in-progress=peach bg, todo=cream bg, blocked=red bg + white fg (alarm
  stays). The hand-darkened light foreground variants from T-38 are deleted.
- Bars/charts/rollup counts on light: `statusColor` returns the RAW palette
  hues (area fills and the chips carry the color; no fg-contrast dependency).
  Rollup count text on light: indigo, with a thin chip per count or plain
  indigo numbers — implementer picks the cleaner frame, records it.
- Single source holds, restated: `statusColor` + `statusStyle` remain the ONLY
  status→visual paths (grep gate stays).
- Measured contrast for every light combination (indigo on cream/peach/olive/
  bronze; white on red) in the report; 120×35 light AND dark frames recorded —
  dark frame byte-identical to pre-change.
- Spec §8 color sentence updated.

## Out of scope
Dark theme changes; layout; config keys; CLI colors; new palette values
(the five given hexes + indigo-ink + alarm red + chrome dim are the whole
vocabulary).

## Gates
- [ ] Dark 120×35 frame byte-identical to pre-change; light frame shows palette-true chips — both recorded with contrast numbers.
- [ ] statusColor/statusStyle still the only status→visual paths (grep); T-38's darkened light variants deleted.
- [ ] just check green (fmt/vet/lint/tests incl. TestStartupPerformance); build ok; spec §8 updated.

## Report
