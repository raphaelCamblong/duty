# Tasks

The whole model in a breath: one task is one markdown file, sized for a single
sitting. Tracks are folders; a `BOARD.md` in each keeps the order. No database,
no daemon, no git dependency — just the files, the folders, and one binary.

A handful of rules run through everything:

- **The file is the truth.** The board only reflects it — order and grouping. On
  any disagreement the file wins, and duty flags the drift rather than guessing.
- **File and board move together.** Every state change writes the task and its
  board row in one command, so they can't drift apart on their own.
- **The TUI only watches.** It never writes; edits go through the CLI or your
  editor, and the viewer shows them instantly.
- **Directions in, report out.** A task file carries what to do and, later, a
  record of what happened — never code. Every append dates itself: a
  `### 2006-01-02 15:04` heading, plus ` — status` when `--status` is given,
  so nobody has to hand-write a date.
- **Agent-first CLI.** Quiet on success; a clear non-zero exit and one stderr
  line when something's wrong.

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

- Sections split mechanically on `^## `; gates are a `- [ ]` checklist, and
  tooling counts `[x]` vs `[ ]` for progress (`2/3`).
- **Naming:** `T-NN-<slug>.md`. `NN` is the next integer across the **entire
  tree** — every board, open and archived — zero-padded to two digits, growing
  past 99. Tree-wide uniqueness lets every command take a bare id; a task's
  board is its folder, so `move` never renames. The `slug` derives from the
  title (lowercased, non-alphanumerics → `-`, collapsed, ≤40 chars).
- **Statuses:** `todo | in-progress | done | blocked`. A flat setter — any
  transition is legal. Discipline lives in the lifecycle contract below, not a
  state machine.
- **Claim identity:** an optional `claimed-by: <name>` line names the agent
  currently holding an `in-progress` task. It is machine-owned and absent from
  fresh tasks — `duty get next --claim --as <name>` and `duty status <id>
  in-progress --as <name>` write it (falling back to the `DUTY_AGENT` env var,
  else the claim stays unnamed), and any status leaving `in-progress` clears it.
  The field only ever means "currently holds it": one owner, no history.

:::note[Who owns what]
Frontmatter is machine-owned; the body is yours but CLI-editable. Automated sync
writes (`status`, `create`, `move`, `archive`) touch only the frontmatter and
the board. `set` and `gates` are the sanctioned exception — they edit the body
line-surgically, so every byte outside the target section or checkbox survives.
:::

## Lifecycle

How a task moves from idea to done — the worker's contract:

0. **Author** → `duty create task <title> --body` pipes the whole markdown body
   in one shot; `duty set <id>` bulk-replaces `## ` sections afterward.
1. **Start** → `duty get next --claim` returns the first actionable task and
   marks it `in-progress` in one call (or `duty status <id> in-progress`).
2. **Blocked** → `duty report <id> --status blocked` pipes a report naming
   *exactly* what's missing, flips the status in one locked write, then stop.
3. **Working** → tick gates as they pass (`duty gates check <id> <n>`, or
   `--all` once they all do, or edit the box by hand).
4. **Done** → all gates ticked, `duty report <id> --status done` pipes a report
   — files changed, gate output, deviations and why, follow-ups — and flips the
   status in the same write.
5. Reports **accumulate** — appended, never overwritten.

:::caution
Respect `blocked-by`: don't start a task whose dependencies aren't `done`, and
never guess past a blocker. If a task turns out to be two, finish the stated
scope and name the split in the report — don't expand.
:::
