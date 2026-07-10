---
id: T-17
title: TUI master-detail layout and track naming
status: done
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
- [x] View-model/update tests: panel focus transitions, filter, descend/up,
  preview content for both a task and a track selection.
- [x] Headless `View()` recorded in the report: 120×35 shows header state + two
  panels (task preview and track summary variants); 70×20 falls back to single
  panel.
- [x] `duty track x --title X` and `duty board y --title Y` both create tracks
  (alias covered by a test).
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

**T-17 report (master-detail TUI + track naming).**

Files changed:
- `internal/tui/entry.go` (new): `entry` list items (track/task/section-header), `boardEntries`, `compactDelegate` — 1-line bubbles/list delegate with per-column widths, BubbleZone marks per selectable row, and fuzzy-match rune underlining via `lipgloss.StyleRunes` + `MatchesForItem`.
- `internal/tui/model.go` (rewritten): master-detail `Model` — bubbles `list` (left) + `viewport` preview (right), `focusArea` toggle, filter routing (all keys to the list while typing), header-skipping selection, per-track selection memory, snapshot rebuild keeping filter/selection/scroll, glamour cache per task id cleared on re-scan/resize. New accessors: `PreviewFocused`, `SelectedID`.
- `internal/tui/view.go` (rewritten): header box = breadcrumb + SUBTREE per-status counts + ntcharts bar; left/right bordered panels (accent = focused, dim = blurred); track summary card (totals, rollup, bar, sections with counts, subtree drift); glamour task body; T-15 bottom pane deleted (`preview`/`goalPreview`/`trackPreview`/`geom`/`bodyLines` gone). Single-panel fallback below 80 cols; `enter` opens the preview full-screen.
- `internal/tui/mouse.go` (rewritten): wheel scrolls the hovered panel (BubbleZone `InBounds`; preview glides on the harmonica spring, list moves selection), left click selects a row / focuses the preview, double-click opens/descends.
- `internal/tui/keys.go`: added `tab` (panel) and `/` (filter) bindings; short + full help via bubbles/help.
- `internal/tui/scan.go`: dropped the now-unused `Row.Goal` (T-15 pane plumbing); comment sweep sub-board→track.
- `internal/cli/track.go` (renamed from board.go): `duty track <name> [--title T]` primary, `board` kept via cobra `Aliases`; behavior and exit codes unchanged.
- Naming sweep: `task-system-spec.md` (§1/§2 definition line "A track is a folder; its board defines its state", §4, §6 table row, §7, §8 rewritten for the master-detail layout/keys/mouse/responsive fallback, §9/§10), `internal/app/readme.md.tmpl` + `tests/testdata/readme.md` golden, `duty/README.md`, comments in app/board packages. Package/API identifiers untouched.
- Tests: `tests/tui_test.go` — new `TestMasterDetailLayout` (120×35 track-card + task-preview variants, 70×20 single-panel fallback), `TestPanelFocusAndFilter` (tab/enter/esc focus transitions, fuzzy filter + clear, descend/up with per-track selection memory), mouse tests rewritten to hit BubbleZone zones located from the rendered frame, wheel-over-preview spring test; `tests/cli_test.go` — track/board alias subtest.

Gate tails:
- `go test ./tests/... -coverpkg=./internal/... -count=1` → `ok  github.com/raphaelCamblong/duty/tests  4.6s  coverage: 83.3% of statements in ./internal/...`
- `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

Headless `View()` observations:
- 120×35: header box shows `Board` + `1 in-progress · 1 todo · 1 blocked · 1 done` + stacked ntcharts bar; left panel lists `❯ backend/  Backend  1 blocked · 1 done`, then `Open tasks` with T-01/T-02 rows (status colored, gates column); right panel (track selected) shows the summary card `2 tasks · 1 done`, rollup, bar, `Sections / Open tasks 2`, `no drift`; with T-01 selected the right panel shows `T-01  in-progress` over the glamour-rendered body (Goal text visible).
- 70×20: single panel — header + list only, no preview; `enter` on T-01 replaces the list with the full-screen preview (`T-01  in-progress` + glamour body); `esc` returns. Footer help bar present in all variants.

Deviations:
- Spec §8 updated in the same change (per CLAUDE.md): board-view/bottom-pane/detail-view layout replaced by the master-detail description; wheel smoothing now applies to the preview panel (the list is bubbles-paginated), harmonica retained there.
- `Row.Goal` removed from the scan snapshot — it existed only to feed the deleted T-15 bottom pane; the preview renders the full cached file content instead (still zero file opens on navigation).
- New indirect deps pulled by bubbles/list: `sahilm/fuzzy`, `atotto/clipboard` (go.mod/go.sum).
