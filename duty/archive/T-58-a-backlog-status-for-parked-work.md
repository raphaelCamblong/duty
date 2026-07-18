---
id: T-58
title: A backlog status for parked work
status: done
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
- [x] Scratch tree: backlog task never returned by get next --claim even when
  first in board order; `status <id> todo` makes it claimable immediately.
- [x] TUI fixture with all five statuses: sort order correct, 5-segment bar
  with min-1-cell holding, backlog rendered dim in both theme frames.
- [x] `just check` green; docs updated; round-trip suite green.

## Report

### 2026-07-18 01:56 — done

Added `backlog` as a fifth full-member status. It's settable via `status` and
`report --status`, valid in frontmatter, board rows, and the TSV status column
(contract shape unchanged). `get next`/`--claim` never offer it — actionable
stays "todo with met deps" — so a parked task waits until `duty status <id>
todo` grooms it back in; the claim guard doesn't apply since backlog is never
held.

In the TUI it inks the chrome dim grey in both themes and sorts between blocked
and done, inherited through `rollupOrder` and `statusColor` (single sources).
The min-1-cell bar rule holds with five segments. Waits are unchanged: a backlog
task with unmet deps still shows waits, and a todo blocked-by a backlog dep
counts it unmet.

Files: internal/task/task.go (StatusBacklog + ValidStatus), internal/app/app.go
(error listing), internal/tui/view.go (rollupOrder), internal/tui/theme.go
(statusInk/statusColor). Docs: docs/tasks.md, docs/cli.md.

Tests: TestBacklogNotActionable, TestBacklogWaitsInterplay,
TestStatusSortPlacesBacklogBeforeDone, TestBacklogRendersDim, two 5-segment
BarCells subtests, and a backlog transition in TestRoundTrip.

Deliberately out of scope: get tracks kept its fixed 7-column count shape (no
backlog column added — shape unchanged), and the skill and scaffolded README
template were left as-is per the task's doc list.
