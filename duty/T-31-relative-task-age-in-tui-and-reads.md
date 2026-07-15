---
id: T-31
title: Relative task age in TUI and reads
status: done
blocked-by: []
---

# T-31 — Relative task age in TUI and reads

## Goal
You can see at a glance how fresh every task is: `6m ago` / `2h ago` in the TUI
and in the insightful reads, absolute date once it's old news.

## Read first
`internal/tui/scan.go` (rows already carry file content — mtime comes from the
same visit), `internal/app/get.go` (`TaskInfo`), spec §6 TSV contracts + §8.

## Scope
- **One formatter, one home:** new leaf package `internal/humanize` (zero deps):
  `RelTime(t, now time.Time) string` — `<1m` = `just now`, then `Nm ago`,
  `Nh ago`, `Nd ago`; **above 7 days** switch to the absolute date
  (`2026-07-08`). Both presentation layers (`cli`, `tui`) import it — the rule
  lives once.
- **Data:** `app.TaskInfo` and `app.Row` gain `UpdatedAt time.Time` from
  `fsys.Stat` mtime (same directory visit as the existing read — no extra
  walk); TUI scan `Row` likewise.
- **CLI (the insightful reads):** `get task` / `get next` human output gains an
  `updated:` line; `get tasks` human lines gain a trailing dim-style age
  column. `--agent` TSV: RFC3339 mtime appended as a new TRAILING field on
  `get tasks`, `get task`, `get next` records — trailing so existing parsers
  keep working; spec §6 field lists updated.
- **TUI:** an age column on task rows (dim, right of gates) and in the preview
  header. Keybind `t` toggles it (listed in help); default ON at ≥100 cols,
  OFF below (the toggle still works on small screens). Spec §8 updated.
- Tests: `RelTime` table (boundaries: 59s, 61s, 23h, 25h, 6d23h, 8d), mtime in
  the snapshot/TaskInfo (fixture with a touched file), TUI toggle transition +
  a frame showing/hiding the column, TSV trailing-field shape.

## Out of scope
created/author metadata (mtime only — no frontmatter change); git-based
history; config keys for the threshold; sorting by age.

## Gates
- [x] `RelTime` boundary table green; `get task`/`get next`/`get tasks` show
  ages (human) and RFC3339 trailing field (`--agent`) — verified in a scratch
  tree with a back-dated file (`touch -t`).
- [x] TUI: `t` toggles the column; 120×35 frame shows it, 70×20 hides it by
  default — recorded in the report. `TestStartupPerformance` stays green.
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` 0 issues; `gofumpt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok.
- [x] Spec §6 + §8 updated in the same change.

## Report

Done. Relative task age now surfaces in the reads and the TUI.

New leaf package internal/humanize (zero internal deps): RelTime(t, now) — "just
now" under a minute, "Nm/Nh/Nd ago" up to 7 days, absolute date (2006-01-02)
past a week. Imported by both cli and tui so the rule lives once.

Data: app.TaskInfo and app.Row gained UpdatedAt (time.Time) from fsys.Stat mtime
(best-effort helper App.mtime); TUI scan Row likewise, sourced from the ReadDir
entry's Info() — same directory visit, no extra walk.

CLI: get task / get next human gained an "updated:" line; get tasks human gained
a trailing dim age column (lipgloss faint, auto-stripped when piped). --agent
TSV appends the RFC3339 mtime as a new TRAILING field on get tasks (5→6), get
task and get next (8→9) — existing positional parsers unaffected.

TUI: dim age column right of the gates on task rows + a "· age" segment in the
preview header; keybind "t" toggles it (in the ? help grid); default ON at >=100
cols, OFF below, re-derived on resize until the user toggles. Verified frames:
120x35 shows "just now", 70x20 hides it by default, 70x20 + "t" turns it on.

Spec §6 (field lists) and §8 (row/header/keys) updated in this change.

Gates: go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, coverage
85.9%. golangci-lint run -> 0 issues. gofumpt -l . empty. go vet clean. build ok.
Tests added: RelTime boundary table (59s/61s/23h/25h/6d23h/7d/8d + future),
scan/TaskInfo mtime from a touched file, TUI toggle transitions + on/off frames,
and the --agent trailing-field shape for get task/tasks/next.

Deviation (necessary): the trailing --agent field changes record widths, so the
field-count guards in three existing tests were bumped to the new contract
(cli_reads_test.go 8->9 x2; cli_lifecycle_test.go 5->6, its full-record equality
narrowed to the stable [:5] prefix). No assertion was weakened — the counts are
stricter and the new trailing field has dedicated new coverage. The 't' binding
lives in the FullHelp grid only, not the short bar, to keep the 100-col short-help
line (already at capacity) from truncating "quit" and breaking an existing test.
