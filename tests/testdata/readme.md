# duty/ — the task convention

One file = one task, small enough for one worker (human or agent) in one sitting.
[BOARD.md](BOARD.md) is the index: order top-to-bottom = build order. The task file is
the truth; the board is a projection the `duty` CLI keeps in sync.

## Task file

Frontmatter (`id`, `title`, `status`, `blocked-by`) + sections: `## Goal`,
`## Read first`, `## Scope` (decisions are pre-made — don't re-decide),
`## Out of scope`, `## Gates` (a `- [ ]` checklist; all ticked = done),
`## Report` (append, never overwrite). Author it in one shot — no editor needed:
`create task --body` pipes the whole body in, `set` fills or bulk-replaces sections,
`gates add`/`check` manage the checklist.

Statuses: `todo | in-progress | done | blocked`.

## Commands

| Command | Behavior |
|---|---|
| `duty create task <title>` | New task in the current board (`--slug`, `--blocked-by`, `--section`). `--body` reads the whole task body (`## ` sections) from stdin below a generated H1 — author it in one call. |
| `duty create track <name>` | New track — a folder with its own board — under the current one (`--title`). |
| `duty set <id> [section]` | Replace one section's body from stdin (`set T-07 goal < goal.md`), or every `## ` block at once with no section arg (`set T-07 < body.md`) — line-surgical; a missing section is created. |
| `duty gates <id>` | List a task's gates 1-based (`--agent` for TSV). `gates add <id> <text>...` appends one or more in one write; `gates check`/`uncheck <id> <n>` flip the n-th, or `--all` every gate. |
| `duty status <id> <status>` | Set status in the task file AND its board row. Claiming a task already `in-progress` needs `--force` (take over a stale claim). |
| `duty move <id>` | `--track PATH` moves the task to another track (path from the tree root); `--section NAME` moves its board row under `## <section>`. At least one flag. |
| `duty report <id>` | Append stdin under the task's `## Report`; `--status S` also flips the status (file + board) in the same atomic write — the done/blocked lifecycle endings. |
| `duty archive` | Move every `done` task into its board's `archive/`. |
| `duty delete task <id>` | Remove an open task (`--force` for `done`). |
| `duty get tasks` | Open tasks from the files, with drift flags (`--agent` for TSV). |
| `duty get task <id>` | One task's metadata and file path — not its body (`--agent` for TSV, `--section NAME` for one section's body, `--body` for the whole body). |
| `duty get tracks` | Per-status task counts for every board (`--agent` for TSV). |
| `duty get next` | The first actionable task in board order; empty when nothing is ready (`--agent` for TSV). `--claim` atomically marks it `in-progress` — parallel agents each get a distinct task. |
| `duty tui` | Live board viewer. |

`--in PATH` targets a board by its track path from the tree root (`.` = root), from
anywhere in the tree — on `create task`, `create track`, `get tasks`, `get tracks`,
`get next`, and `archive`. Unknown path → `unknown track "PATH"`.

## Lifecycle → command

0. Author → `duty create task <title> --body` pipes the whole markdown body in one call,
   or `duty set <id>` bulk-replaces `## ` sections from stdin afterward — no editor needed.
1. Start → `duty get next --claim` (the first actionable task, marked `in-progress` in one
   call), or `duty status <id> in-progress` for a task by id.
2. Blocked (missing input, failed dep, unmade decision) → `duty report <id> --status blocked`
   pipes a report naming exactly what's missing and flips the status in one write, then stop.
   Never guess past a blocker.
3. Working → tick gates as they pass (`duty gates check <id> <n>`, or `--all` once they all pass).
4. Done (all gates ticked) → `duty report <id> --status done` pipes a report — files changed,
   gate output tails, deviations (with why), follow-ups deliberately left — and flips the
   status in the same write.
5. Respect `blocked-by`: don't start a task whose dependencies aren't `done`.
6. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.

## What stays your judgment

Filling Goal/Scope/Gates when authoring a task, ticking gates honestly, report prose,
and respecting `blocked-by` — the tooling checks none of it.
