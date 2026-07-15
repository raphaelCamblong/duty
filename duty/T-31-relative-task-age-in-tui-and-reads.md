---
id: T-31
title: Relative task age in TUI and reads
status: todo
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
- [ ] `RelTime` boundary table green; `get task`/`get next`/`get tasks` show
  ages (human) and RFC3339 trailing field (`--agent`) — verified in a scratch
  tree with a back-dated file (`touch -t`).
- [ ] TUI: `t` toggles the column; 120×35 frame shows it, 70×20 hides it by
  default — recorded in the report. `TestStartupPerformance` stays green.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` 0 issues; `gofumpt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok.
- [ ] Spec §6 + §8 updated in the same change.

## Report
