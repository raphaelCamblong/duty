---
id: T-08
title: CLI archive, delete, list, agent output
status: todo
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
- [ ] Tests: archive is idempotent (second run = no-op, byte-identical tree) and
  archives into the task's own board; delete guards `done` without `--force`; list
  flags a hand-desynced row without touching it; `--agent` field order asserted
  exactly.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
