---
id: T-58
title: A backlog status for parked work
status: todo
blocked-by: []
---

# T-58 — A backlog status for parked work

## Goal
Tasks can exist without being offered to anyone: `backlog` joins the status
vocabulary — visible, counted, and skipped by `get next` until promoted.

## Read first
`internal/task` (status constants, ValidStatus), `internal/app/get.go`
(actionable = todo + met deps — backlog must simply not qualify),
`internal/tui` (statusColor/rollupOrder — single sources, the new status
inherits everywhere), docs/tasks.md statuses.

## Scope
- New status `backlog` (full member: settable via `status`/`report --status`,
  valid in frontmatter, board rows, TSV status column — contract shape
  unchanged).
- **Semantics**: `get next`/`--claim` never return backlog tasks (actionable
  stays "todo with met deps"); promotion is explicit (`duty status <id>
  todo`). The claim guard doesn't apply (backlog isn't held by anyone).
- **Display**: color = the chrome dim grey (parked, not alarming) in both
  themes; sort order becomes in-progress, todo, blocked, backlog, done;
  rollups, track bars, and the header distribution inherit via rollupOrder/
  statusColor (verify the min-1-cell bar rule still holds with 5 segments).
- Waits annotation interplay: a backlog task with unmet deps shows waits too
  (it's still true); a todo task blocked-by a BACKLOG dep counts as unmet
  (backlog isn't done) — no rule change needed, verify by test.
- Docs: tasks.md status line + one lifecycle sentence (backlog → todo when
  groomed); cli.md status mentions; skill UNCHANGED (statuses aren't
  enumerated there — --help covers it).
- Tests: get next skips backlog; promotion; sort position; 5-segment bar;
  round-trip with a backlog transition; TSV carries the value.

## Out of scope
Auto-promotion rules; per-status boards/sections; TUI mutations; renaming
existing statuses.

## Gates
- [ ] Scratch tree: backlog task never returned by get next --claim even when
  first in board order; `status <id> todo` makes it claimable immediately.
- [ ] TUI fixture with all five statuses: sort order correct, 5-segment bar
  with min-1-cell holding, backlog rendered dim in both theme frames.
- [ ] `just check` green; docs updated; round-trip suite green.
