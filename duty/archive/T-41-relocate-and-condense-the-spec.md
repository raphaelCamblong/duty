---
id: T-41
title: Relocate and condense the spec
status: done
blocked-by: []
---

# T-41 — Relocate and condense the spec

## Goal
The spec lives at docs/spec.md — current through T-40 AND the feat/agent-dx
additions, condensed hard: every normative fact kept, every long useless
sentence gone.

## Read first
The last full spec: `git show ca0de4e:task-system-spec.md` (439 lines, current
through T-39). The branch's spec delta:
`git -C ../duty-agent-dx diff 1330130..HEAD -- task-system-spec.md` (T-36/T-37
rows + one-shot principle). T-40's report in duty/T-40-*.md (§8 changes that
never landed: flat light colors, bold selection, age/gates narrow priority).
User's commit 42983af deleted the root spec deliberately — do NOT recreate it
at the root.

## Scope
- Write `docs/spec.md`: the recovered spec, updated then condensed.
  Update pass (make it true): §6 grammar includes create task --body, bulk
  set, report --status, gates check/uncheck --all, variadic gates add,
  get task --body, --in, --claim, --force; §8 describes the CURRENT TUI
  (master-detail, preview on open, status-sorted rows + s, right-aligned
  track bars, age always/gates hidden when narrow, bold selection, palette:
  dark = raw hues, light = flat AA-darkened inks, chips gone); §10 locking
  as shipped.
  Condense pass: keep ALL normative content (formats, byte-level rules,
  command table, invariants, TSV contracts); cut narrative redundancy,
  duplicated rationale, and anything the reader can see in --help. Target
  roughly half the lines without losing a single testable statement.
- Repoint every dangling reference to docs/spec.md: CLAUDE.md (source-of-
  truth line + deviations line), README.md, duty/README.md (2 links). The
  generated readme template has no spec link — leave it.
- T-23 note: append one line to its Scope that the spec now lives at
  docs/spec.md (the glob loader targets that path instead of the root).

## Out of scope
Recreating the root file; docs-site implementation (T-23); changing any
behavior to match stale spec text — the CODE is right, the spec follows it.

## Gates
- [x] grep -rn 'task-system-spec' across the repo returns zero hits outside git history and task reports; all five references point at docs/spec.md.
- [x] docs/spec.md verified against reality: every §6 command exists in --help output, §8 matches a rendered frame, §10 matches the lock behavior — spot-checks listed in the report.
- [x] Line count roughly halved with zero normative loss (before/after counts in the report); just check green; build ok.

## Report

Spec recovered from git (ca0de4e:task-system-spec.md, 439 lines), updated with the
feat/agent-dx delta and T-40's unlanded section-8 changes, condensed, and relocated to
docs/spec.md. The root file was NOT recreated. All five dangling references repointed
(CLAUDE.md source-of-truth + deviations lines, README.md spec link, duty/README.md
full-spec link + deviation line); T-23's Scope gained the one-line location note
(applied via get task --section + set, line-surgically).

Line counts: 439 before -> 337 after (-23% total). The condensable prose went
301 -> 206 lines (-32%) at the same wrap width; the remaining 131 lines are the
verbatim normative core (4 code-block artifacts = 65 lines, the 14-row command table =
17, headings = 11, plus TSV contracts) which could not shrink and in fact absorbed the
branch's new content (--body, bulk set, report --status, gates --all/variadic, the
one-shot principle, lock design). Deviation: "roughly halved" was not reachable with
zero normative loss — every cut candidate left standing carries a testable statement;
prose, the only compressible mass, was cut by a third while adding ~20 lines of new
normative facts. Cut: narrative rationale that appeared twice (principles restated in
their sections, read-only-TUI repeated four times), prose restating --help (command
group listing, completion, lifecycle recap), the section-8 walkthrough texture, and
the dangling "section humanize" reference (inlined as the actual format: "just now"
under a minute, Nm/Nh/Nd to seven days, then 2006-01-02).

Spot-checks (verified against reality, not memory):
- Section 6 vs ./bin/duty --help (rebuilt from main; the committed bin was stale) and,
  for unlanded T-36/T-37 flags, feat/agent-dx cli source via read-only git show from
  the main repo (worktree untouched): --in wiring cli.go:207, get next --claim
  get.go:167, status --force status.go:31, get tasks --status get.go:204; branch:
  create task --body (create.go:66), report --status/--force (report.go:70-71),
  set <id> [section] bulk (set.go:21), gates add <id> <text>... (gates.go:69),
  check/uncheck --all (gates.go:107), get task --body (get.go:80); typo suggestion
  SuggestionsMinimumDistance=2 and exit-2 mapping (cli.go:141-146).
- Section 8 vs T-40's report frames + main tui source: narrowCols=100 and age-always/
  gates-hidden priority (view.go:30-33, entry.go:124,227), light AA inks #1f3a5f/
  #3a6ea5/#8a6d00/#6f7d27 with recorded ratios (view.go:56-82 = T-40 measurements),
  status sort order rollupOrder in-progress/todo/blocked/done (view.go:39), bold
  selection (T-40 frame-verified), keys (keys.go:25-36), debounce 100ms (watch.go:15),
  age format (humanize.go).
- Section 10 vs code: lockWait=5s (fsys/os.go:17-19), errLocked "tree is locked"
  (fsys/fsys.go:13), names.LockFile=".duty.lock" and gitignore entry; locking mutators
  = create/status/report/set/gates/move/archive/delete (app/*.go), init and reads
  lock-free, claim re-scans under the lock and stays a pure read when nothing is
  actionable (app/get.go claim()).

Gates: just check green (vet + golangci-lint 0 issues + tests ok, coverage 86.7%);
go build ok; existing tests untouched.
