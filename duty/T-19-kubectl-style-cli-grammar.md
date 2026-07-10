---
id: T-19
title: kubectl-style CLI grammar
status: todo
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
- [ ] Every command in the table above works end-to-end in a scratch tree; old
  spellings (`duty create <title>`, `duty track`, `duty board`, `duty link`,
  `duty delete <id>`) fail with exit 2 except hidden `list`.
- [ ] Full suite green after migration (`go test ./tests/... -coverpkg=./internal/...
  -count=1`) — invariants round-trip runs on the NEW spellings.
- [ ] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [ ] Spec §6, README, and readme template/golden all show only the new grammar.

## Report
