---
id: T-55
title: Computed views show dependency waits
status: done
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
- [x] Fixture with a dependency chain: `get tasks` shows waits on exactly the
  unmet tasks; annotation disappears the moment the dependency is done
  (test through cli.Run).
- [x] TUI frame shows the dim wait annotation (recorded in report);
  TestStartupPerformance green.
- [x] One predicate implementation (grep — no duplicated unmet-deps logic);
  `just check` green; docs updated.

## Report

### 2026-07-16 14:47 — done

Computed views now show dependency waits; BOARD.md untouched (recorded rejection honored).

One predicate: `app.UnmetDeps(deps, statusOf)` in internal/app/get.go with the
`depMet` (done||archived) rule — the sole unmet-deps logic. get next's actionable
walk, get tasks (List), and the TUI scan all route through it; the two callers
differ only in the status source (file reads vs the snapshot), never the rule.

Files:
- internal/app/get.go — UnmetDeps + depMet + depStatus/depStatuses; Dep type,
  TaskInfo.Deps; GetTask/GetNext fill per-dep statuses; actionable uses unmetDeps.
- internal/app/list.go — Row.Waits, threaded the tree root through boardRows/taskRow.
- internal/cli/get.go — get tasks trailing dim `waits T-01,T-03`; get task
  blocked-by annotates each id `(status)`; --agent TSV unchanged.
- internal/tui/scan.go — annotateWaits pass (snapshot-only, no extra reads;
  archived/done dep absent from the open set counts as met); Row.Waits.
- internal/tui/entry.go — task row dim `waits …` beside the status.
- internal/tui/view.go — waitsTag; preview header strikes met blocked-by ids
  (StyleRunes), keeping the "blocked-by <id>" text contiguous.
- docs/tasks.md, docs/cli.md, docs/tui.md — annotations documented.

Tests (tests/, black-box via cli.Run + a TUI frame):
- cli_waits_test.go — TestGetTasksWaits (waits on exactly the unmet task,
  shrinks then vanishes as deps finish), TestGetTaskBlockedByStatuses (per-dep
  status annotation; --agent field 7 stays plain T-01,T-02).
- tui_waits_test.go — TestTUIWaitAnnotation (scan marks only unmet deps; frame
  shows the annotation), TestTUIArchivedDepCountsMet (archived dep met, no reads).

Gates: `just check` green (lint 0 issues incl. funlen 35). TestStartupPerformance
green — 1.99ms best of 5, the wait pass reuses the already-parsed snapshot.

Note: the TUI treats a truly-missing blocked-by id as met (it can't see archived
files without extra reads); the CLI, which resolves files, flags it `(missing)`
and as an unmet wait. This divergence is confined to a typo'd dep id.

### 2026-07-16 14:52 — done

Applied feedback-round simplify: folded depStatuses into fillDeps (single-caller indirection, behavior identical) and deleted it. Skipped the waitsCell/waitsTag dup — both reviewers agree there is no clean shared home (cli/tui cannot import each other; app returns data not formatted strings), so it stays the architecture accepted cost.
