---
id: T-15
title: TUI goal preview and track status rollups
status: done
blocked-by: []
---

# T-15 — TUI goal preview and track status rollups

## Goal
Scrolling the board previews the selected task's Goal without any extra file I/O,
and track (sub-board) rows show real per-status state instead of a bare done count.

## Read first
`task-system-spec.md` §8, `CLAUDE.md` (TUI rules), `internal/tui/scan.go` and
`view.go` as built by T-10/T-11.

## Scope
- **Goal in the snapshot:** during the existing scan (files are already read once
  there), extract each task's `## Goal` section text into the view model. Zero file
  opens on navigation — the preview reads the snapshot only.
- **Preview panel:** board view gains a bottom preview pane showing the selected
  task's Goal (2–4 lines, dim rounded border, ellipsis-truncated, adaptive colors).
  Hidden automatically when the terminal is too short or a track row is selected
  (tracks preview their per-status summary instead). Resizes gracefully.
- **Track rollups:** replace `n/m done` on sub-board rows with compact per-status
  counts, zero-count statuses omitted, each count colored with its status color
  (e.g. `1 in-progress · 2 todo · 4 done`), consistent with the header ntcharts bar.
- Update `task-system-spec.md` §8 (board view + sub-board row description) in the
  same change — the spec tracks reality.
- Tests: view-model exposes goal text and per-status counts (fixture tree);
  update-transition tests unchanged; headless `View()` render shows the preview
  pane and rollup line.

## Out of scope
Any file-format change; any mutation path; preview for archived tasks; config keys.

## Gates
- [x] View-model tests: goal extracted during scan, per-status counts correct on a
  fixture with all four statuses.
- [x] Headless `View()` at 110×30 shows the preview pane for a task row and the
  rollup counts on a track row; at a short height the preview hides — recorded in
  the report.
- [x] `go test ./tests/... -coverpkg=./internal/... -count=1` green; `gofmt -l .`
  empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

Implemented the Goal preview pane and per-status track rollups.

Files changed:
- internal/task/task.go: added pure `Section(content, heading)` — trimmed body of a
  "## <heading>" section, stops at the next "## ".
- internal/tui/scan.go: `Row.Goal` (captured via task.Section during the existing
  scan — zero extra file opens on navigation); `Board.Counts`/`Sub.Counts` per-status
  subtree tallies aggregated in `link` alongside Done/Total.
- internal/tui/view.go: bottom preview pane (dim rounded border) wired into `geom`
  between body and footer so the mouse hit-test still agrees; task rows preview the
  word-wrapped, ellipsis-truncated Goal (<=4 lines, adaptive dim), sub-board rows
  preview a per-status summary; auto-hides when the terminal can't keep minBodyLines
  task rows on screen. `subLine` now renders `statusRollup` (per-status counts, each in
  its status color, zero counts omitted, dim middot separators) replacing "n/m done".
- task-system-spec.md §8: board view + preview pane + rollup description updated.
- tests/task_test.go: TestSection. tests/tui_test.go: TestGoalAndCountsInSnapshot and
  TestPreviewPaneAndRollups over a four-status fixture.

Gate output:
- go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, coverage 82.7%.
- gofmt -l . empty; go vet ./... clean; go build -o bin/duty ./cmd/duty ok.
- Headless View() at 110x30: task row shows "Ship the alpha milestone..." in the
  preview pane; track row shows "1 blocked . 1 done" rollup plus a "2 total . ..."
  summary pane; at 110x8 the preview auto-hides (goal text absent).

Deviations: rollup status order is [in-progress, todo, blocked, done] to reproduce the
Scope's example string "1 in-progress . 2 todo . 4 done" exactly; colors match the
ntcharts header bar per the Scope. The current board's footer keeps "n/m done" (only
sub-board rows were in scope). Empty subtrees render a dim "empty".

Follow-ups: none.
