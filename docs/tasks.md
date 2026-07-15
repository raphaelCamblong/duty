# Tasks

A few ideas run through everything here. One file is one task — plain text, sized
for one sitting. The task file is the truth and the board just reflects it (order
and grouping); on any disagreement the file wins and duty flags the drift instead
of guessing. Every state change writes the task and its board row together, in one
command, so they can't drift apart on their own. The TUI only watches — it never
writes; edits go through the CLI or your editor. A task file carries directions
and, later, a report of what happened — never code. The CLI is built agent-first:
quiet when things go well, a clear non-zero exit and one line on stderr when they
don't. No database, no daemon, no git dependency — just the files, the folders,
and the binary.

## The task file

```markdown
---
id: T-01
title: Short imperative title
status: todo            # todo | in-progress | done | blocked
blocked-by: []          # task ids that must be done first
---

# T-01 — Short imperative title

## Goal
1–2 lines: the outcome, not the steps.

## Read first
Docs and files the worker must read before writing anything.

## Scope
What to build/change. Decisions are PRE-MADE here; the worker doesn't re-decide.

## Out of scope
What NOT to touch. YAGNI with teeth.

## Gates
- [ ] Runnable command or observable check.
- [ ] Another one. All boxes ticked = done.

## Report
(appended on completion/blockage — accumulates; the task file is the record)
```

- Sections split mechanically on `^## `; gates are a `- [ ]` checklist, tooling counts
  `[x]` vs `[ ]` for progress (`2/3`).
- **Frontmatter is machine-owned; the body is user-owned but CLI-editable.**
  *Automated* sync writes (`status`, `create`, `move`, `archive`) touch only
  frontmatter and the board. *Explicit* section edits (`set`, `gates`) are the
  sanctioned exception and stay line-surgical: every byte outside the target section
  or checkbox survives; the body is never re-rendered from a model.
- **Naming:** `T-NN-<slug>.md`. `NN` = next integer across the **entire tree** — all
  boards, open AND archived — zero-padded to two digits, growing past 99. Tree-wide
  uniqueness lets every command take a bare id; a task's board is its folder, so
  `move` never renames. `slug` derives from the title (lowercase, non-alphanumerics →
  `-`, collapsed, ≤40 chars).
- **Statuses:** `todo | in-progress | done | blocked`. Flat setter — any transition is
  legal; discipline lives in the lifecycle contract below, not a state machine.

## Lifecycle

The worker's contract: how a task moves from idea to done.

0. **Author** → `duty create task <title> --body` pipes the whole markdown body in one
   shot; `duty set <id>` bulk-replaces `## ` sections from stdin afterward.
1. **Start** → `duty get next --claim` returns the first actionable task and marks it
   `in-progress` in one call (or `duty status <id> in-progress` by id).
2. **Blocked** → `duty report <id> --status blocked` pipes a report naming *exactly*
   what's missing, flips the status in one locked write, then stop — never guess past
   a blocker.
3. **Working** → tick gates as they pass (`duty gates check <id> <n>`, `--all` once
   they all pass, or edit the box by hand).
4. **Done** (all gates ticked) → `duty report <id> --status done` pipes a report —
   files changed, gate output tails, deviations (with why), follow-ups left — and
   flips the status in the same write.
5. Reports **accumulate** — appended, never overwritten. Respect `blocked-by`: don't
   start a task whose dependencies aren't `done`.
6. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.
