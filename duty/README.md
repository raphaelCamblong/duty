# duty/ — the task convention

One file = one task, small enough for one worker (human or agent) in one sitting.
[BOARD.md](BOARD.md) is the index: order top-to-bottom = build order. The task file is
the truth; the board is a projection. A track is a folder; its board defines its state
(`duty create track <name>` creates one). Full spec: [../task-system-spec.md](../task-system-spec.md).

## Task file

Frontmatter (`id`, `title`, `status`, `blocked-by`) + sections: `## Goal`, `## Read
first`, `## Scope` (decisions are pre-made — don't re-decide), `## Out of scope`,
`## Gates` (a `- [ ]` checklist; all ticked = done), `## Report` (append, never overwrite).
Author it in one shot — no editor needed: `duty create task <title> --body` pipes the whole
markdown body in at once, `duty set <id> [section]` fills one section or bulk-replaces every
`## ` block from stdin, `duty gates add/check/uncheck <id>` manage the checklist.

Statuses: `todo | in-progress | done | blocked`.

## Lifecycle → command

0. Author → `duty create task <title> --body` pipes the whole markdown body in one call,
   or `duty set <id>` bulk-replaces `## ` sections from stdin afterward — no editor involved.
1. Start → `duty get next` (the first actionable task), then `duty status <id> in-progress`.
2. Blocked (missing input, failed dep, unmade decision) → `duty status <id> blocked`
   + pipe a report naming exactly what's missing (`duty report <id>`), then stop.
   Never guess past a blocker.
3. Working → tick gates as they pass (`duty gates check <id> <n>`).
4. Done (all gates ticked) → `duty status <id> done` + pipe a report: files changed,
   gate output tails, deviations (with why), follow-ups deliberately left.
5. Respect `blocked-by`: don't start a task whose dependencies aren't `done`.
6. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.

Reading state: `duty get next` (first actionable task), `duty get task <id>` (`--section
NAME` prints one section's body), `duty get tasks`, `duty get tracks`, `duty gates <id>`
(add `--agent` for TSV). Cleanup: `duty archive`.

## What stays your judgment

Filling in report prose, ticking gates honestly, and flagging spec bugs — if the code
must deviate from `task-system-spec.md`, fix the spec in the same change.
