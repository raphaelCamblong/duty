---
id: T-10
title: TUI live board viewer
status: done
blocked-by: [T-05, T-08]
---

# T-10 — TUI live board viewer

## Goal
`duty tui` per spec §8: navigable board + detail views, keyboard driven, refreshing
live from the filesystem.

## Read first
`task-system-spec.md` §8, §7 (theme/editor); `CLAUDE.md` TUI rules (pure update
logic, bubbles before hand-rolling, lipgloss only).

## Scope
- `internal/tui`: a pure scan → view-model layer (boards, section/row order from each
  `BOARD.md`, truth + gate counts from files, drift badges), then the Bubble Tea
  model/update/view on top.
- Board view: breadcrumb (H1 titles), sub-board rows with live counts
  (`Backend  3/7 done`), sections, rows with colored status (todo dim, in-progress
  yellow, blocked red, done green), gate progress, `⚠` drift badge.
- Detail view: task markdown via glamour inside `bubbles/viewport`.
- Keys: `j/k`, `enter` (descend / open), `esc` (up / close), `e` → `$EDITOR` from
  config (suspend/resume), `q`.
- Live refresh: fsnotify watch per directory, ~100 ms debounce, full re-scan on any
  event, re-walk to add watches when directories appear.
- Theme from config (`auto|dark|light`). Read-only: zero writes from the TUI.
- New deps: bubbletea, bubbles, lipgloss, glamour, fsnotify.

## Out of scope
Mouse, harmonica, ntcharts, `?` help footer (all T-11); any mutation path.

## Gates
- [x] View-model tests on a fixture tree in `tests/`: ordering matches the boards,
  counts and drift computed from files, archived ignored.
- [x] Update-transition tests: key messages produce the expected navigation states
  (no terminal needed).
- [x] Observed manually and noted in the report: `duty tui` renders the real tree,
  and an external `duty status` + an `$EDITOR` save each appear without a keypress.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

Built `internal/tui` as three layers: `scan.go` (filesystem → `Snapshot` view model:
boards keyed by root-relative path, sections/rows in `BOARD.md` order, truth/gates
from the files, drift flags `board says <s>` / `no row` / `no file` / `unparsable
file`, subtree counts rolled up for sub-board rows), `model.go` + `keys.go`
(value-semantics Bubble Tea model; pure `Update`; bubbles/key named bindings with
help text ready for T-11's help footer; `tea.ExecProcess` for `$EDITOR` with
suspend/resume; detail = glamour in `bubbles/viewport`, re-rendered on resize and on
every re-scan so an open task refreshes live), `view.go` (lipgloss only:
AdaptiveColor pairs for auto/dark/light, rounded-border header with H1-title
breadcrumb, colored statuses — todo dim / in-progress yellow / blocked red / done
green — gate progress, aligned `⚠` drift column, ellipsis truncation via
`x/ansi.Truncate`, dim footer status line with subtree counts + drift total,
selected line kept in a scroll window). `watch.go`: fsnotify watch per directory,
100 ms debounce coalescing bursts into a buffered channel, re-walk before every
notification so new directories (sub-boards) get watches live. `run.go` wires
FindRoot + config (user < project) + `lipgloss.SetHasDarkBackground` for explicit
themes + `tea.WithAltScreen`. Zero writes anywhere in the package.

Files changed: `internal/tui/{scan,model,view,keys,watch,run}.go` (new),
`internal/cli/tui.go` (new) + `tui` case in `internal/cli/cli.go`,
`internal/board/sections.go` (new small helper: `board.Sections`/`board.Row`, pure
read-only section+row parser — the scan needed row order and the board package owns
the format; tested in `tests/board_test.go`), `internal/task/task.go` (new small
helper `task.Body`: strips frontmatter for the detail render since the header
already shows id/status; tested in `tests/task_test.go`), `tests/tui_test.go` (new).

Gate tails: `go test ./tests/... -coverpkg=./internal/...` →
`ok github.com/raphaelCamblong/duty/tests 1.746s coverage: 81.2% of statements`;
`gofmt -l .` empty; `go vet ./...` clean. golangci-lint not installed.

Manual gate, verified headlessly (no interactive terminal available in this
environment): (1) rendered `Model.View()` for the real `duty/` tree at 110×26 via a
throwaway harness (deleted, not committed) — breadcrumb box, all 11 rows with
colored statuses and gate counts (`T-10 … todo 0/4`), footer `9/11 done`; fixture
frames (board with sub-board line `backend/ Backend 1/1 done`, drift badge
`⚠ board says done`, detail view) logged by `TestViewRendersHeadless -v` and
eyeballed. (2) The no-keypress refresh path is proven end to end by
`TestWatcherRefresh`: an external `duty status` write → debounced tick → re-scan
shows the new status; `duty board` + `duty create` inside the brand-new directory
tick too (re-walk added the watch); an in-place file write (an `$EDITOR`-save-shaped
event) ticks and the re-scan shows the new gate count. (3) `bin/duty tui` from the
repo exits 1 with one-line `could not open a new TTY…` when no tty — clean CLI
behavior.

Deviations: none from the spec §8 scope of this task (mouse/harmonica/ntcharts/`?`
footer are T-11 per the task split). CLAUDE.md's "nothing imports `cli` or `tui`"
is read as the inward dependency rule — `duty tui` dispatch requires `cli` → `tui`,
as this task's scope mandates. Follow-ups deliberately left: T-11 polish layer.
