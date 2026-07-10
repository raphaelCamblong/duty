---
id: T-22
title: TUI polish night pass
status: done
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
- [x] Update tests: `r` triggers a re-scan; breadcrumb click navigates; empty
  track and empty-filter frames render the hints.
- [x] `TestStartupPerformance` still passes (no renderer/queries while browsing).
- [x] Frame audit at the five sizes recorded in the report, no ragged frames.
- [x] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.

## Report

## T-22 report — TUI polish night pass

Scope delivered in full; no stubs.

**Files changed**
- internal/tui/keys.go — added `Refresh` (`r`) binding; compacted `enter` hint to `↵` so the short-help bar fits 100 cols; listed refresh in ShortHelp + FullHelp.
- internal/tui/model.go — wired `r` to a re-scan; `findRowBoard` for the preview header's track title; list no-items renamed to "match/matches" and styled dim.
- internal/tui/view.go — clickable BubbleZone breadcrumb (crumbChain + crumbZone, dim `›` separators); centered empty-state hints (fresh tree / empty track) + reused list no-items line for no-filter-matches; preview header `id · status · gates n/m · track title` with dim blocked-by + drift; helpView now truncates per line (backstop for a bubbles/help width quirk).
- internal/tui/mouse.go — breadcrumb-segment clicks jump to the ancestor track.
- internal/tui/entry.go — anySelectable helper (empty vs no-match detection).
- internal/tui/scan.go — Row now carries BlockedBy from frontmatter.
- tests/tui_test.go — TestManualRefresh, TestBreadcrumbClickNavigates, TestEmptyStates (fresh tree / empty track / no-match), TestFrameAudit (five sizes, per-line width assertion).
- task-system-spec.md §8 — breadcrumb click, `r`, empty states, preview header.

**Gate tails**
- `go test ./tests/... -coverpkg=./internal/...` → ok, coverage 84.7%.
- TestStartupPerformance → best of 5 = 2.08ms (< 100ms; no glamour/query while browsing).
- TestFrameAudit → 120x35, 100x28, 80x24, 70x20, 60x16 all pass; no line exceeds terminal width in browse or preview frames.
- `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

**Deviations / notes**
- Two library quirks found and worked around (not regressions): (1) bubbles/help stops truncating when the ellipsis itself won't fit, overflowing narrow frames — fixed with an ansi.Truncate backstop per help line; (2) bubbles/list auto-clears a filter that matches nothing on `enter`, so the no-match hint is rendered from the visible-vs-total entry distinction (shown while typing) rather than via the list's post-accept state.
- `enter` hint shown as `↵` (was `enter`) to keep the full short-help bar visible without truncation at 100 cols — sanctioned polish.

**Follow-ups left (none blocking):** none.
