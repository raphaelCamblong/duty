---
id: T-07
title: CLI status, link, report, move
status: todo
blocked-by: [T-06]
---

# T-07 — CLI status, link, report, move

## Goal
The four mutation commands from spec §6, each maintaining the sync invariant in one
shot.

## Read first
`task-system-spec.md` §6 (`status`/`link`/`report`/`move` rows), §3–§5; `CLAUDE.md`.

## Scope
- `status <id> <status>` — frontmatter `status:` line + board row cell; reject unknown
  statuses; reject archived ids (read-only).
- `link <id> <section>` — move the row under `## <section>` (created above the footer
  if absent); prune any section left empty.
- `report <id>` — append stdin under `## Report` (heading created once); refuse empty
  stdin.
- `move <id> <board-path>` — path relative to tree root (`.` = root board); rename the
  file into the target board (same filename), drop source row + prune, append to the
  target's `Open tasks` (or `--section`), status preserved.

## Out of scope
archive/delete/list (T-08), TUI, any format change.

## Gates
- [ ] Black-box tests per command: file+board both updated in one call; unknown
  status rejected; empty-stdin report rejected; move preserves status and filename
  and leaves both boards internally consistent; move to a non-existent board errors.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
