# duty/ — the task convention

One file = one task, small enough for one worker (human or agent) in one sitting.
[BOARD.md](BOARD.md) is the index: order top-to-bottom = build order. The task file is
the truth; the board is a projection the `duty` CLI keeps in sync.

## Task file

Frontmatter (`id`, `title`, `status`, `blocked-by`) + sections: `## Goal`,
`## Read first`, `## Scope` (decisions are pre-made — don't re-decide),
`## Out of scope`, `## Gates` (a `- [ ]` checklist; all ticked = done),
`## Report` (append, never overwrite).

Statuses: `todo | in-progress | done | blocked`.

## Commands

| Command | Behavior |
|---|---|
| `duty create <title>` | New task in the current board (`--slug`, `--blocked-by`, `--section`). |
| `duty board <name>` | New sub-board under the current board (`--title`). |
| `duty status <id> <status>` | Set status in the task file AND its board row. |
| `duty link <id> <section>` | Move the board row under `## <section>`. |
| `duty report <id>` | Append stdin under the task's `## Report`. |
| `duty move <id> <board-path>` | Move a task to another board (path from the tree root). |
| `duty archive` | Move every `done` task into its board's `archive/`. |
| `duty delete <id>` | Remove an open task (`--force` for `done`). |
| `duty list` | Open tasks from the files, with drift flags (`--agent` for TSV). |
| `duty tui` | Live board viewer. |

## Lifecycle → command

1. Start → `duty status <id> in-progress`.
2. Blocked (missing input, failed dep, unmade decision) → `duty status <id> blocked`
   + pipe a report naming exactly what's missing (`duty report <id>`), then stop.
   Never guess past a blocker.
3. Working → tick gate checkboxes in the task file as they pass.
4. Done (all gates ticked) → `duty status <id> done` + pipe a report: files changed,
   gate output tails, deviations (with why), follow-ups deliberately left.
5. Respect `blocked-by`: don't start a task whose dependencies aren't `done`.
6. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.

## What stays your judgment

Filling Goal/Scope/Gates when authoring a task, ticking gates honestly, report prose,
and respecting `blocked-by` — the tooling checks none of it.
