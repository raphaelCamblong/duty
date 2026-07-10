---
id: T-20
title: "Agent reads: get task, tracks, next"
status: done
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
- [x] Scratch-tree checks: `get task` both forms; `get tracks` counts match a
  hand-built tree; `get next` picks the first unblocked todo in board order and
  skips a todo blocked by an in-progress dependency.
- [x] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.
- [x] Spec §6 and README updated in the same change.

## Report

Implemented `get task`, `get tracks`, `get next` — logic in internal/app returning
data, formatting in internal/cli.

Files changed:
- internal/app/get.go (new): TaskInfo/TrackInfo + GetTask, GetTracks, GetNext and
  helpers (buildTaskInfo, trackInfo, nextInBoard/actionable, depsDone/depDone,
  taskStatuses, archivedCount, relBoard). GetNext walks tree.Boards from the current
  board (current-board rows first via board.Sections order, then sub-tracks depth-first
  in scan order), returns the first todo whose blocked-by are all done; archived deps
  count as done (ErrArchived short-circuits), nil when nothing is actionable.
- internal/cli/get.go: get task/tracks/next subcommands + human aligned key:value and
  --agent TSV formatters. Group usage now `duty get <task|tasks|tracks|next> [args]`.
- task-system-spec.md §6: three new command rows + agent-output TSV formats (get task /
  get next 8-field record; get tracks 7-field record) and the get-next empty-exit-0 rule.
- internal/app/readme.md.tmpl + tests/testdata/readme.md (golden, lockstep): three
  reader rows in the command table; lifecycle step 1 now leads with `duty get next`.
- duty/README.md: lifecycle + reading-state line updated to name the new readers.
- tests/cli_reads_test.go (new): get task (human/agent, sub-track, gates, blocked-by,
  unknown/missing id), get tracks (agent fixed columns, human counts, current-board
  scoping), get next (board order, blocked-by chains incl. in-progress dep, archived-dep
  -as-done, file-truth-over-board, empty case + empty tree, agent form).
- tests/cli_test.go: bare-get usage expectation. tests/roundtrip_test.go: TestReadsNeverWrite
  now exercises get task/tracks/next too (reads never write a byte).

Gate tails:
- go test ./tests/... -coverpkg=./internal/... -count=1 → ok, coverage 83.3%.
- gofmt -l . empty; go vet ./... clean; go build -o bin/duty ./cmd/duty ok.
- Scratch-tree: get task both forms correct; get tracks counts match a hand-built tree
  (own-tasks-only + archived); get next picks first unblocked todo in board order and
  skips a todo blocked by an in-progress dependency.

Deviations / judgment calls (none contradict the spec):
- get tracks and get next are scoped recursively from the current board (root renders as
  "."), mirroring get tasks — the spec's "including the root (.)" reads as the current
  board.
- Human get task shows "none" for an empty blocked-by; get next skips a board row whose
  file is missing (drift) rather than erroring; a blocked-by id that resolves nowhere
  counts as not-done (keeps the task blocked). These are robustness choices the spec
  leaves open.
- Enriched the README/template command table with the three readers (beyond the required
  lifecycle mention) so the agent-facing doc stays complete.
