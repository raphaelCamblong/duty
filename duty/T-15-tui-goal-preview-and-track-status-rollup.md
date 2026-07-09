---
id: T-15
title: TUI goal preview and track status rollups
status: todo
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
- [ ] View-model tests: goal extracted during scan, per-status counts correct on a
  fixture with all four statuses.
- [ ] Headless `View()` at 110×30 shows the preview pane for a task row and the
  rollup counts on a track row; at a short height the preview hides — recorded in
  the report.
- [ ] `go test ./tests/... -coverpkg=./internal/... -count=1` green; `gofmt -l .`
  empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report
