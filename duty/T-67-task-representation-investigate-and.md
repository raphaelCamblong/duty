---
id: T-67
title: "Task representation: investigate and redesign"
status: in-progress
claimed-by: fable
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
- IMPLEMENTATION: staged, contracts frozen. Frozen: file formats, the
  round-trip byte-identity invariant, line-surgical writes, exit codes, and
  every existing TSV token/field in its position. NOT frozen: the eight
  divergences the investigation found — the same tree answering differently
  per surface is the defect this task exists to remove. Each divergence
  resolves to one deliberate rule, enumerated in the report with its
  rationale; new TSV fields append only in documented positions; tests and
  docs update in the same change. (This replaces the original byte-frozen
  gate: the investigation showed byte-freezing would force one model to
  reproduce contradictory semantics — see report.)
- Explicitly NOT: a compatibility shim over the old paths, or a quick
  adapter. Old assembly paths get deleted, not wrapped.

## Out of scope
File-format changes; TUI visual restyling; performance regressions
(TestStartupPerformance holds); the stored archived footer's staleness
(mutation-side, noted for a follow-up).

## Gates
- [x] Investigation findings + chosen design + rejected alternatives
  recorded in the report before implementation commits.
- [ ] The old parallel assembly paths are gone (grep-provable); one loading
  path feeds list/get/next/tui/watch.
- [ ] Every deliberate behavior change is enumerated in the report and
  covered by an updated or new test; everything not enumerated is
  byte-identical (round-trip + salted-board + golden suites green).
- [ ] Full suite green, TestStartupPerformance green; depth and identifier
  scans clean over the rewritten paths (T-66's five deferrals resolved).
- [ ] just check green; docs (internals.md, cli.md, tui.md) updated.

## Report

### 2026-07-19 21:00

INVESTIGATION (two full code maps, producer + consumer side)

What exists today: four independent file-to-struct assemblers build
overlapping task views — app taskRow (list), app buildTaskInfo (get/next),
app boardStates (watch), tui readTasks+joinRow (scan). Around them: three
board-order rules (list appends strays at the end in ReadDir order, the TUI
sorts them into the default section, get next skips rowless files
entirely), three drift models (list knows 2 classes, the TUI 4, watch 0),
two dependency oracles with opposite semantics for a deleted id (CLI:
missing = blocks; TUI: absent = met — the T-55 divergence, live and
comment-documented in scan.go), two status tallies (get tracks counts 4
statuses and silently drops backlog; the TUI counts all 5), three live
archived counters plus a stored footer count nothing reads back, and
duplicated title-fallback / mtime / file-enumeration rules.

Cost: get tasks resolves every blocked-by id with a full-tree WalkDir plus
a fresh parse, per dependent, no memoization — O(tasks x deps x tree).
get tracks fully parses every task file to extract one field. claim
parses the winner three times. Same tree state, different answers:
an unparsable task file fails get tasks entirely, renders as a drift row
in the TUI, and is silently dropped by watch.

Agent-visibility gaps: get tasks --agent carries neither claimed-by nor
waits; an orchestrator must call get task per row to learn either.

DESIGN — one loader, one joined model, thin projections (all in
internal/app; no new package, app IS the query layer)

Types: TreeView (root + boards in walk order) > BoardView (dir, path,
title, sections in board order, archived count, archived views when
loaded) > SectionView > TaskView carrying the three truths joined once:
file truth (id, title, status, blocked-by, claimed-by, gates, content,
mtime), board truth (row file, row status, typed Drift: none / no-row /
status / no-file / bad-file), computed truth (waits, deps) — plus one
in-memory dep oracle (open statuses + archived ids from archive ReadDir
names; archive file contents load only behind the existing toggle).

Load(scope) walks once from the root, reads each index once, parses each
task once, computes drift and waits from memory. List, GetTask, GetNext,
GetTracks, Snapshot, and tui Scan become projections over TreeView; the
TUI keeps only its display extras (subs, rollups, bars). Mutations stay
line-surgical and untouched — reads get one model, writes keep surgery.

Complexity drops to one walk + one parse per file per invocation, dep
checks become map lookups.

REJECTED ALTERNATIVES
1. internal/view package below app — a layer only app would import;
   layering for its own sake.
2. Incremental cache invalidated by fsnotify — duty-scale trees reload
   under the 100ms startup budget; complexity unjustified; watch stays
   full-rescan.
3. Extract shared helpers, keep the four assemblers — the lazy patch;
   leaves the divergence class alive.
4. Byte-frozen unification — one model cannot reproduce contradictory
   semantics (a missing dep cannot be both met and unmet); gate amended
   accordingly.
5. TUI keeps a mirror Row struct over TaskView — pure duplication; the
   TUI consumes app types directly.

ENUMERATED BEHAVIOR CHANGES (each rationale'd, each tested, docs updated)
1. A blocked-by id that resolves nowhere blocks everywhere — the TUI
   adopts the CLI rule (safety: a deleted dep must not silently unblock).
2. An unparsable task file no longer fails get tasks; it degrades to a
   bad-file drift row (agent-first robustness; matches the TUI).
3. A board row pointing at no file appears in get tasks with no-file
   drift, status from board truth (fulfils cli.md's drift promise).
4. One stray rule: rowless files sort into the default section
   everywhere; a rowless unblocked todo becomes reachable by get next.
5. get tracks gains a backlog column (todo, in-progress, done, blocked,
   backlog, archived) — backlog tasks are no longer invisible in counts.
6. get tasks --agent appends claimed-by and waits fields (positions 7-8;
   existing fields 1-6 unchanged).
7. A duplicate board row for one file renders once everywhere (the TUI
   previously showed it twice).
8. watch keeps skipping unparsable files — now as a documented projection
   rule (transient editor half-saves should not emit events), not an
   accident.
Dep-oracle rule stays: met = done or archived; backlog blocks; board-only
rows never satisfy a dep (no file truth).

Implementation stages: S1 loader + model with tests (no consumer moves);
S2 app producers become projections, old assembly deleted, cli formatters
+ docs updated; S3 tui scan over the loader, D1 flip, goldens; S4
internals.md + re-run depth/identifier scans (T-66's five deferrals) +
full gate.
