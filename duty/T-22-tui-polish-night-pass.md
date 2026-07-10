---
id: T-22
title: TUI polish night pass
status: todo
blocked-by: [T-21]
---

# T-22 — TUI polish night pass

## Goal
The TUI feels finished: helpful empty states, informative preview header,
clickable breadcrumb, manual refresh — every frame looks intentional at any size.

## Read first
`internal/tui` as it stands after T-18 (preview on open, single lazy renderer —
do NOT regress the startup path: no glamour work and no terminal queries while
browsing), `task-system-spec.md` §8.

## Scope
- **Empty states:** an empty track shows a centered dim hint (`no tasks yet —
  duty create task "…"`); a filter with no matches shows bubbles/list's
  no-items state styled to match; an empty tree (fresh init) says so.
- **Preview header:** opened task shows `id · status (colored) · gates n/m ·
  track title`; blocked-by ids listed dim when present.
- **Breadcrumb navigation:** each breadcrumb segment is a BubbleZone — click
  jumps to that track (mouse-only shortcut for `esc` chains).
- **Manual refresh:** `r` re-scans immediately (the watcher stays; `r` is
  reassurance), listed in help.
- **Frame audit:** render headless frames at 120×35, 100×28, 80×24, 70×20,
  60×16; fix anything ragged (truncation, border math, help overflow); record
  the frames in the report.
- Spec §8 updated where behavior changed (breadcrumb click, `r`, empty states).

## Out of scope
New panels or layout changes; mutations from the TUI; performance work beyond
not regressing T-18's gates; CLI.

## Gates
- [ ] Update tests: `r` triggers a re-scan; breadcrumb click navigates; empty
  track and empty-filter frames render the hints.
- [ ] `TestStartupPerformance` still passes (no renderer/queries while browsing).
- [ ] Frame audit at the five sizes recorded in the report, no ragged frames.
- [ ] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.

## Report
