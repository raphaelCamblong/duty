---
id: T-67
title: "Task representation: investigate and redesign"
status: todo
blocked-by: []
---

# T-67 — Task representation: investigate and redesign

## Goal
One mature representation of "a task as the system sees it" — replacing the
parallel structs and duplicated assembly paths that grew organically — chosen
by investigation, not by patching what exists.

## Read first
The known evidence: task.Task (domain frontmatter), app.Row (list),
app.TaskInfo (get/next), the TUI scan's own Row/Board/Sub, and watch's
Snapshot/boardStates each re-assemble overlapping task views via separate
walk-and-read code (boardRows, nextInBoard, buildTaskInfo, tui Scan,
boardStates). T-55 already recorded a real divergence bred by this
duplication (CLI vs TUI disagree on a missing blocked-by id). docs/
internals.md + cli.md for the exposed-feature vision the design must serve.

## Scope
- INVESTIGATION FIRST (its findings go in this task's report before any
  code): map every task-view struct, every assembly path, every consumer;
  list concrete duplications, divergences, and inefficiencies (repeated
  reads/walks per command).
- DESIGN (recorded here before implementation): the orchestrator writes the
  chosen architecture — expected shape is a single canonical loaded model
  (tree → boards → tasks, carrying file truth + board row + computed
  drift/waits/order once) with list/next/get/tui-scan/watch-diff becoming
  queries over it — but the investigation decides; alternatives considered
  get one line each on why not.
- IMPLEMENTATION: staged, behavior-frozen. CLI outputs byte-identical, TUI
  frames identical, watch events identical — the invariants suite and every
  golden referee it. Test edits only for direct internal-call syntax,
  listed.
- Explicitly NOT: a compatibility shim over the old paths, or a quick
  adapter. Old assembly paths get deleted, not wrapped.

## Out of scope
CLI surface changes; file-format changes; TUI visual changes; performance
regressions (TestStartupPerformance holds).

## Gates
- [ ] Investigation findings + chosen design + rejected alternatives
  recorded in the report before implementation commits.
- [ ] The old parallel assembly paths are gone (grep-provable); one loading
  path feeds list/get/next/tui/watch.
- [ ] Full suite green, byte-frozen outputs, TestStartupPerformance green.
- [ ] just check green; docs/internals.md updated to describe the model.
