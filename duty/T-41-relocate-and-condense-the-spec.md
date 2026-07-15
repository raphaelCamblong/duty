---
id: T-41
title: Relocate and condense the spec
status: todo
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
- [ ] grep -rn 'task-system-spec' across the repo returns zero hits outside git history and task reports; all five references point at docs/spec.md.
- [ ] docs/spec.md verified against reality: every §6 command exists in --help output, §8 matches a rendered frame, §10 matches the lock behavior — spot-checks listed in the report.
- [ ] Line count roughly halved with zero normative loss (before/after counts in the report); just check green; build ok.

## Report
