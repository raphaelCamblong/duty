---
id: T-20
title: "Agent reads: get task, tracks, next"
status: todo
blocked-by: [T-19]
---

# T-20 — Agent reads: get task, tracks, next

## Goal
Agents (and humans) can ask the three questions that matter: what is this task,
what state is every track in, and what should I work on next.

## Read first
`task-system-spec.md` §6 (grammar + agent-output contract as rewritten by T-19),
`internal/app`.

## Scope
- `duty get task <id> [--agent]` — metadata, not the body (the file path is
  printed; readers `cat` it). Human: aligned `key: value` lines — id, title,
  status, track, blocked-by, gates `n/m`, path. `--agent`: one TSV record
  `id, track-path, status, title, gates-done, gates-total, blocked-by
  (comma-joined), path`.
- `duty get tracks [--agent]` — one line per track including the root (`.`):
  path, title, per-status counts (todo/in-progress/done/blocked, own tasks
  only) and archived count. `--agent`: TSV `path, title, todo, in-progress,
  done, blocked, archived` — fixed column order is the contract.
- `duty get next [--agent]` — the first actionable task: walk the current
  board's rows in board order (build order is priority), then sub-tracks
  depth-first in scan order; emit the first `todo` whose `blocked-by` are all
  `done` (archived counts as done). Output = same shape as `get task`. No
  actionable task → no output, exit 0 (empty means nothing to do — document it).
- All logic in `internal/app` (returns data); `internal/cli` formats. Spec §6
  gains the three rows; README/template mention `get next` in the lifecycle
  ("Start → `duty get next`").
- Tests: each command human + `--agent`; `next` respects blocked-by chains,
  board order, archived-dependency-counts-as-done, and the empty case.

## Out of scope
Mutations; body printing in `get task`; help polish (T-21); TUI (T-22).

## Gates
- [ ] Scratch-tree checks: `get task` both forms; `get tracks` counts match a
  hand-built tree; `get next` picks the first unblocked todo in board order and
  skips a todo blocked by an in-progress dependency.
- [ ] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.
- [ ] Spec §6 and README updated in the same change.

## Report
