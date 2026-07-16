---
id: T-55
title: Computed views show dependency waits
status: todo
blocked-by: []
---

# T-55 — Computed views show dependency waits

## Goal
A reader can tell actionable from waiting at a glance: tasks with unmet
`blocked-by` show a dim `waits T-NN` in `get tasks` and the TUI — without
touching the board file format.

## Read first
User feedback (2026-07-16): three blocked-by'd todos render identically to
the one actionable task; the reporter proposed projecting into BOARD.md —
REJECTED here (recorded decision): a wait-state cell would make status
computed, force cross-file row rewrites on every dependency completion, and
grow the drift surface. The board stays 3 columns of truth (order + status);
richness lives in computed views. `internal/app/get.go` (depsDone — the
actionable logic to reuse), `internal/app/list.go`, the TUI scan/entry.

## Scope
- `get tasks` human: tasks whose `blocked-by` has unmet deps (per the same
  rule `get next` uses — archived counts as done) gain a dim trailing
  `waits T-01,T-03`. `--agent` TSV unchanged (blocked-by is already col 7 —
  the reporter noted parsers have what they need).
- TUI task rows: same dim annotation (near the status), and the preview
  header already lists blocked-by — verify it distinguishes met/unmet (strike
  or dim the met ones if cheap, else leave).
- `get task` human: `blocked-by:` line annotates each id with its status
  (`T-01 (done)`, `T-03 (in-progress)`).
- One computation, one home: the unmet-deps predicate lives once in app
  (shared with get next's actionable walk), consumed by list/TUI scan.
- docs/tasks.md blocked-by paragraph mentions the annotations.

## Out of scope
BOARD.md format changes (decision above); auto-status flips (blocked-by
stays advisory, per the lifecycle contract); new TSV fields.

## Gates
- [ ] Fixture with a dependency chain: `get tasks` shows waits on exactly the
  unmet tasks; annotation disappears the moment the dependency is done
  (test through cli.Run).
- [ ] TUI frame shows the dim wait annotation (recorded in report);
  TestStartupPerformance green.
- [ ] One predicate implementation (grep — no duplicated unmet-deps logic);
  `just check` green; docs updated.
