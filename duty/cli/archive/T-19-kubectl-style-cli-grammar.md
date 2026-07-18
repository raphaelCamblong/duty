---
id: T-19
title: kubectl-style CLI grammar
status: done
blocked-by: []
---

# T-19 — kubectl-style CLI grammar

## Goal
The CLI reads verb → resource, kubectl-style: `duty create task`, `duty create
track`, `duty get tasks`, `duty delete task` — one grammar, no mixed forms.

## Read first
`task-system-spec.md` §6, `internal/cli` and `internal/app` as they stand,
kubectl's grammar for the model (create/get/delete on resources; terse verb-only
commands where the resource is unambiguous).

## Scope
The new surface (a CLEAN BREAK — old spellings removed, tests migrate; the user
sanctioned this):
- `duty create task <title> [--slug S] [--blocked-by ID…] [--section NAME]`
- `duty create track <name> [--title T]` (top-level `track`/`board` commands removed)
- `duty get tasks [--status S] [--agent]` — the old `list`; keep `list` as a
  HIDDEN alias so agent muscle-memory survives one more cycle.
- `duty move <id> [--track PATH] [--section NAME]` — absorbs `link` (removed):
  `--track` relocates the file+row to another track (status preserved, target
  section = `--section` or Open tasks); `--section` alone moves the row within
  the current board; at least one flag required.
- `duty delete task <id> [--force]` (replaces `delete <id>`)
- Unchanged: `init`, `status <id> <s>`, `report <id>`, `archive`, `tui` — the
  agent hot path stays terse (kubectl has verb-only commands too).
- Structure: cobra parent commands `create`, `get`, `delete` with resource
  subcommands; `internal/app` methods renamed/merged to match (Move absorbs
  Link). Contract otherwise frozen: quiet success, one-line lowercase errors,
  exit 0/1/2, `--agent` TSV unchanged.
- Docs in the same change: spec §6 table + §5 lifecycle mentions, `duty/README.md`,
  `internal/app/readme.md.tmpl` + its golden, help strings.

## Out of scope
New read commands (`get task/tracks/next` — T-20); help polish, groups, completion
(T-21); TUI (T-22); any file-format change.

## Gates
- [x] Every command in the table above works end-to-end in a scratch tree; old
  spellings (`duty create <title>`, `duty track`, `duty board`, `duty link`,
  `duty delete <id>`) fail with exit 2 except hidden `list`.
- [x] Full suite green after migration (`go test ./tests/... -coverpkg=./internal/...
  -count=1`) — invariants round-trip runs on the NEW spellings.
- [x] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [x] Spec §6, README, and readme template/golden all show only the new grammar.

## Report

kubectl-style grammar landed: create task/track, get tasks (hidden list alias), move <id> --track/--section (absorbs link), delete task <id>; init/status/report/archive/tui untouched.

Files changed:
- internal/cli: cli.go (missingCommandError + newGroupCmd, exit-2 mapping), create.go (verb + task/track resources), get.go (new; get tasks + hidden list + TSV/human formatting), move.go (flag-driven), delete.go (verb + task); link.go, list.go, track.go removed.
- internal/app: move.go (Move absorbs Link as moveRow; targetBoard speaks "track"), board.go -> track.go (CreateBoard -> CreateTrack); link.go removed.
- tests: cli_test.go (dispatch table covers every old spelling at exit 2; TestCreateTrack), cli_mutate_test.go (TestMoveSection/TestMoveTrack), cli_lifecycle_test.go (get tasks + hidden-alias subtest, delete task), roundtrip_test.go (round-trip and salted mutations on new spellings), tui_test.go fixtures.
- docs: spec sect. 4/6/9, duty/README.md, internal/app/readme.md.tmpl + tests/testdata/readme.md golden.

Gate tails: gofmt -l . empty; go vet ./... clean; go build ok; go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, coverage 83.1%. Scratch-tree run: all table commands green; create <title>/track/board/link/delete <id> all exit 2; list still exits 0 and is hidden from help.

Deviations: dropped the expired "Until the CLI exists" section from duty/README.md while updating its command spellings (it self-expired at T-08 and contradicted the CLI-only sync contract). Old positional "move <id> <path>" exits 1 (usage) rather than 2 - it is a bad argument to a live command, not an unknown command; the gate list did not name it.

Follow-ups left deliberately: "duty create" usage line and cobra's "Usage: duty create [flags]" help polish belong to T-21; get task/tracks/next to T-20.
