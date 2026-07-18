---
id: T-08
title: CLI archive, delete, list, agent output
status: done
blocked-by: [T-07]
---

# T-08 — CLI archive, delete, list, agent output

## Goal
Close the lifecycle: archive, delete, and the reading side (`list`, drift flags,
`--agent` TSV) from spec §6.

## Read first
`task-system-spec.md` §6 (`archive`/`delete`/`list` rows + the agent-output
paragraph); `CLAUDE.md`.

## Scope
- `archive` — every `status: done` task in the current board and below: rename into
  its OWN board's `archive/` (created if missing), drop row, prune, rewrite that
  board's footer count. Idempotent; nothing-to-archive is a clean no-op.
- `delete <id> [--force]` — refuse `done` without `--force`; remove file, drop row,
  prune.
- `list [--status S]` — recursive from the current board, truth from the files:
  `id  status  title`, sub-board path prefix when not local; `⚠ board says …` drift
  flag on disagreement or missing row. Never auto-heals.
- `--agent` on list (long-only, no shorthand): TSV
  `id<TAB>board-path<TAB>status<TAB>title<TAB>drift` (drift empty, `board=<status>`,
  or `no-row`); no padding, no color, no badges.

## Out of scope
TUI, locking, board deletion, healing drift.

## Gates
- [x] Tests: archive is idempotent (second run = no-op, byte-identical tree) and
  archives into the task's own board; delete guards `done` without `--force`; list
  flags a hand-desynced row without touching it; `--agent` field order asserted
  exactly.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

Files changed:
- `internal/cli/archive.go` (new) — `runArchive` walks `tree.Boards` from the current
  board down; for each board, `archiveBoard` collects its `status: done` task files
  (direct children only, via `os.ReadDir` + the new `tree.IsTaskFile`), drops their rows,
  prunes, renames the files into that board's own `archive/` (`os.MkdirAll` first), and
  rewrites the footer count from a post-rename count of `archive/`. A board with nothing
  done is left untouched — this is what makes a second run byte-identical.
- `internal/cli/delete.go` (new) — `runDelete` resolves the id (rejects archived ids via
  the shared `resolveOpen`), refuses `status: done` unless `--force`, computes the
  dropped+pruned board content before removing the file (so a `DropRow` failure leaves
  the file in place), then removes the file and writes the board.
- `internal/cli/list.go` (new) — `runList` walks `tree.Boards` from the current board,
  reads each board's task files directly (`tree.IsTaskFile`), parses each one, and cross
  references `board.FindRow`/`board.RowStatus` to compute drift. Renders either the human
  line (`[prefix ]id  status  title[  ⚠ board says …]`, prefix `"sub/board/ "` matching
  the spec's `backend/ T-12 …` example) or, with `--agent` (long-only bool flag, no
  shorthand defined), the exact TSV `id<TAB>board-path<TAB>status<TAB>title<TAB>drift`
  (drift empty / `board=<status>` / `no-row`). `list` never writes.
- `internal/cli/cli.go` — dispatch cases for `archive`, `delete`, `list`; added a shared
  `unknownStatusErr` helper (both `status` and `list --status` had the identical
  one-liner) and switched `status.go` to use it.
- `internal/tree/tree.go` — added `IsTaskFile(name string) bool`, exporting the existing
  `taskNN` filename regex so `cli` can enumerate task files directly inside a board
  directory without re-deriving the convention. Tested in `tests/tree_test.go`
  (`TestIsTaskFile`).
- `internal/board/board.go` — added `RowStatus(row string) (string, bool)`, extracting
  the status-cell parsing already used inside `SetRowStatus` so `list` can read a row's
  status without mutating it. Tested in `tests/board_test.go` (`TestRowStatus`).
- `tests/cli_lifecycle_test.go` (new) — `TestArchive`, `TestDelete`, `TestList`,
  covering: archiving into the task's *own* board (root vs. a `backend` sub-board, each
  gets its own `archive/` and footer count), non-done tasks and empty trees left
  untouched, idempotent second run (full tree snapshot compared byte-for-byte),
  archive scoped to "current board and below" (a sub-board run doesn't touch a sibling
  parent), delete's `done`/`--force` guard and archived/unknown-id/argument validation,
  list's sub-board prefix, `--status` filter, drift flagging on both a hand-desynced
  status cell and a hand-dropped row (asserting neither the task file nor the board
  changes), and `--agent`'s exact 5-field TSV order for both an in-sync and a drifted
  record.

Gate command tails:
```
$ go test ./tests/... -coverpkg=./internal/...
ok  	github.com/raphaelCamblong/duty/tests	0.587s	coverage: 86.1% of statements in ./internal/...
$ gofmt -l .
(empty)
$ go vet ./...
(clean)
```
(`golangci-lint` is not installed in this environment — skipped per CLAUDE.md's "if
installed".)

Deviations: none from the spec; no spec changes needed.

Smoke test (per the task's closing instruction): from the repo root, `bin/duty list`
against this project's own `duty/` tree printed all eleven tasks (`T-01`..`T-07` as
`done`, `T-08`..`T-11` as `todo`) with no drift flags — the board is in sync — and
`bin/duty list --agent` printed the matching 5-field TSV, `board-path` `.` for every
row (all currently at the tree root), trailing empty drift field. Read-only smoke test;
the project's own `duty/` tree was not otherwise touched (no `archive`/`delete` run
against it).

Follow-ups deliberately left: none — `archive`/`delete`/`list` are scoped exactly to
this task; the master round-trip invariant test (create → … → delete → archive,
tree-hash identical) is explicitly T-09's gate, not this one's, so it isn't added here.
