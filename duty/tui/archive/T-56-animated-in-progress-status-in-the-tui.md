---
id: T-56
title: Animated in-progress status in the TUI
status: done
blocked-by: []
---

# T-56 — Animated in-progress status in the TUI

## Goal
`in-progress` breathes: a small animated spinner next to the status, powered
by bubbles/spinner — alive when work is happening, silent when nothing is.

## Read first
`internal/tui/model.go` (Update loop, where ticks integrate), `view.go`
(statusStyle — the glyph goes beside the status text), bubbles/spinner docs
(already a dependency), the perf guard TestStartupPerformance.

## Scope
- ONE shared `spinner.Model` on the TUI model (MiniDot or Dot — pick what
  reads at one cell), tinted with the in-progress status color (theme-aware:
  peach on dark, the blue ink on light).
- Every visible in-progress task row shows the spinner glyph beside its
  status text; the preview header too when the open task is in-progress.
- Tick lifecycle is the heart of it: the spinner's tick runs ONLY while at
  least one in-progress row exists in the current snapshot — no in-progress
  work anywhere = zero ticks, zero re-renders, zero battery. Re-arm on
  re-scan when in-progress appears; stop when it disappears. Filtering/
  descending that hides them may keep ticking (snapshot-level check is
  enough — don't over-engineer visibility).
- Frame audit sizes still clean (the glyph adds one cell — verify truncation
  math); TestStartupPerformance green (no spinner tick during the browse
  benchmark when the fixture has no in-progress tasks — and add a fixture
  variant proving ticks stop after the last in-progress task completes).
- docs/tui.md one line; spec-of-record §8 sentence.

## Out of scope
Animating anything else (bars, done states); configurable spinner styles;
per-row spinner phases (one shared phase is correct and calm).

## Gates
- [x] Update-transition tests: tick arms when a snapshot gains an in-progress
  task, stops (no scheduled tick) when the last one leaves; glyph present in
  the rendered row and preview header (frames in report).
- [x] TestStartupPerformance green; frame audit green (no ragged lines with
  the extra cell).
- [x] `just check` green; docs updated.

## Report

### 2026-07-16 18:17 — done

Implementation complete (agent stalled at final verification; orchestrator
finished the run). One shared MiniDot spinner tinted with the in-progress
status color, glyph on rows + preview header; tick loop arms only when the
snapshot holds an in-progress task and halts itself when the last one leaves
(TestSpinnerTickLifecycle proves both directions; TestSpinnerGlyphOnInProgressRow
pins the frames). Golden dark/light frames updated for the one glyph cell —
the sanctioned change. just check green: gofumpt clean, vet clean,
golangci-lint 0 issues, full suite green, TestStartupPerformance green.
