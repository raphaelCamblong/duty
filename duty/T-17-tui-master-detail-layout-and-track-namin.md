---
id: T-17
title: TUI master-detail layout and track naming
status: todo
blocked-by: []
---

# T-17 — TUI master-detail layout and track naming

## Goal
The TUI becomes a master-detail workspace — tracks and tasks on the left, a live
preview on the right, the current track's total state on top — and "track" becomes
the one user-facing word for a nested board folder.

## Read first
`task-system-spec.md` §8, `CLAUDE.md` (TUI rules), `internal/tui` as it stands
(T-10/T-11/T-15), bubbles `list` component docs (delegates, built-in fuzzy filter).

## Scope
- **Layout** (lipgloss `JoinHorizontal`/`JoinVertical`, all adaptive colors):
  - **Header:** breadcrumb (track path by H1 titles) + the current track's SUBTREE
    state: per-status counts in status colors + the ntcharts distribution bar.
  - **Left panel** (~38% width, min 30 cols): `bubbles/list` with a custom compact
    delegate — sub-tracks first (name + colored per-status rollup), then tasks under
    their section headers (id, title, status, gates, drift badge). Built-in fuzzy
    filtering on `/`. Selection drives the preview.
  - **Right panel:** preview of the selection. Task → glamour-rendered body in a
    viewport (lazy render on selection change, cached until the next re-scan;
    zero file opens — the snapshot already holds content). Track → summary card:
    title, per-status counts, distribution bar, its sections with counts, drift
    count. Replaces T-15's bottom pane (delete it).
  - **Footer:** `bubbles/help` hints (short + `?` full).
- **Keys:** `j/k` move, `enter` descend into a track / focus the preview on a task,
  `esc` unfocus / up one track, `tab` toggle panel focus, `/` filter, `e` `$EDITOR`,
  `?` help, `q` quit. **Mouse:** BubbleZone — click selects (left) or focuses
  (right), double-click opens/descends, wheel scrolls the hovered panel.
- **Responsive:** below ~80 cols fall back to single-panel (left only; `enter`
  opens the preview full-screen — the pre-T-17 behavior). Graceful resize.
- **Track naming:** "track" is the user-facing word for a folder-with-BOARD.md.
  `duty track <name> [--title T]` becomes the primary command with `board` kept as
  a working alias (cobra Aliases — behavior otherwise frozen). Sweep user-facing
  wording: spec (§2/§4/§8 "sub-board" → track, one definition line: *a track is a
  folder; its board defines its state*), `duty/README.md`, generated readme
  template, TUI labels, help/usage strings. Package/API identifiers stay as-is.
- Live refresh, watcher, and read-only invariants unchanged.

## Out of scope
File-format or CLI-output changes (list TSV etc. untouched); any mutation path;
renaming `BOARD.md` itself; touching `future.md`.

## Gates
- [ ] View-model/update tests: panel focus transitions, filter, descend/up,
  preview content for both a task and a track selection.
- [ ] Headless `View()` recorded in the report: 120×35 shows header state + two
  panels (task preview and track summary variants); 70×20 falls back to single
  panel.
- [ ] `duty track x --title X` and `duty board y --title Y` both create tracks
  (alias covered by a test).
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report
