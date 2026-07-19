# CLI

`duty` is verb-first, kubectl-style: `create`, `get`, and `delete` take a
resource (`task` / `track`); every other command is a bare verb. Whatever
writes keeps the task file and its board row in sync in one atomic step, under a
tree-wide lock — so a swarm of agents never corrupts a board.

It's quiet when it works — with one exception: scaffolding commands (`init`,
`create track`, `skill install`) print what they made and where, since that's
the only way to learn it; every lifecycle mutation (`status`, `report`,
`gates`, `move`, `archive`, `delete`, `set`) stays exactly as silent as before.
An error is one lowercase line on stderr and a non-zero exit; a missing or
unknown command exits `2` and suggests the closest match. `duty --version`
prints the build version.

The commands below are grouped by role — the same four groups `duty --help`
shows.

:::note[The guarantee]
Every writer is line-surgical and atomic: it changes only the target line or
cell and writes through a temp-file rename, so a board is never half-written and
never re-rendered from a model. A full round-trip — create → status → report →
move → move back → delete → archive — leaves the tree byte-identical. Drift
between a file and its board row is surfaced, never silently healed.
:::

## Author

### init

`duty init [title]` — Scaffold a `duty/` tree right here: a `BOARD.md`, the
convention `README.md`, and an `archive/`. Refuses if you're already inside a
tree. Prints where it landed, git-init style.

```sh title="Start a board"
duty init "Q3 roadmap"
# initialized duty tree in /abs/path/to/duty
```

### create task

`duty create task <title>` — Add a task to the current board. The next id
(`T-NN`) is picked across the whole tree, and the file plus the board row are
written together. Prints one line, `<id><TAB><path>` — the id you need next
and the absolute file path, same shape for a human or an agent.

```sh title="A task that waits on another"
duty create task "Add auth" --blocked-by T-03
# T-04	/abs/path/to/duty/T-04-add-auth.md
```

Flags:
- `--slug S` — override the filename slug (default: derived from the title).
- `--blocked-by ID` — a task that must be done first; repeat for several.
- `--section NAME` — board section for the new row (default `Open tasks`).
- `--in PATH` — target another board by its track path (`.` = root).
- `--body` — read the whole markdown body from stdin instead of the skeleton.

:::tip
`--body` is the agent write path: pipe a whole task in one call. Feed it
markdown, not JSON — it splices byte-for-byte.
:::

### create track

`duty create track <name>` — Make a nested track: a folder with its own
`BOARD.md` and `archive/`, and a bullet appended to the parent's board. The
name must be `[a-z0-9-]+`. Prints `<name><TAB><path>`, symmetric with `create
task`.

```sh title="A backend track"
duty create track backend --title "Backend work"
# backend	/abs/path/to/duty/backend
```

Flags:
- `--title T` — track title (default: the name).
- `--in PATH` — create it under another board.

### set

`duty set <id> [section]` — Replace a task section from stdin, line-surgically:
everything outside the section survives. Name a section to replace that one;
give no name and stdin is one or more `## ` blocks, each replaced or created in
order, in one write.

```sh title="Rewrite one section"
duty set T-07 goal < goal.md
```

## Work

### status

`duty status <id> <status>` — Set a task's status — `todo`, `in-progress`,
`done`, `blocked`, or `backlog` — in both its file and its board row. Every
transition is free except claiming a task already `in-progress`, refused so a
live claim is never silently stolen — the refusal names who holds it. Moving to
`in-progress` records the claimer (`--as`, else `$DUTY_AGENT`, else unnamed); any
other status clears the claim.

```sh title="Start a task, claiming it"
duty status T-07 in-progress --as sonnet-2
```

Flags:
- `--force` — take over a task already `in-progress`.
- `--as NAME` — record NAME as the claimer when moving to `in-progress` (falls
  back to `$DUTY_AGENT`).

### report

`duty report <id>` — Append stdin under the task's `## Report` heading, created
once and then accumulating. With `--status`, flip the status in the same write —
the atomic "done" or "blocked" ending.

```sh title="Report and mark done in one write"
duty report T-07 --status done < report.txt
```

Flags:
- `--status S` — also set the status (file + board) in the same write.
- `--force` — with `--status in-progress`, take over an existing claim.
- `--as NAME` — with `--status in-progress`, record NAME as the claimer (falls
  back to `$DUTY_AGENT`).

### gates

`duty gates <id>` — List a task's gates, 1-based (`1 [x] build passes`).

```sh title="List the gates"
duty gates T-07
```

Flags:
- `--agent` — TSV: `index`, `done`, `text`.

### gates add

`duty gates add <id> <text>...` — Append one or more gates, in order, in one
write (creating `## Gates` if absent).

```sh title="Add two gates"
duty gates add T-07 "build passes" "tests green"
```

### gates check / uncheck

`duty gates check <id> <n>` and `duty gates uncheck <id> <n>` — Tick or untick
the n-th gate (1-based), or `--all` of them in one write.

```sh title="Tick every gate"
duty gates check T-07 --all
```

Flags:
- `--all` — flip every gate at once, no `<n>`.

### move

`duty move <id>` — Move a task to another track, to another board section, or
reorder its row within its board — at least one flag. Across tracks the file is
renamed into the target folder (ids don't encode tracks) and its status is
preserved.

```sh title="Move to the backend track"
duty move T-07 --track backend --section "Open tasks"
```

Board order is priority, so `move` also sets it. `--top`, `--before`, and
`--after` relocate the row line verbatim — the same bytes, moved. They're
mutually exclusive, but combine with `--track`/`--section`, applying after the
relocation. `--before`/`--after` take a task id; the row lands next to that
task's row and adopts its section. The reference must be in the same board — a
reference elsewhere errors, telling you to `move --track` it here first. This is
the one write that touches only the board, never the task file.

```sh title="Make a task the next thing to pick up"
duty move T-07 --top
duty move T-07 --after T-03
```

Flags:
- `--track PATH` — target track path from the tree root (`.` = root board).
- `--section NAME` — target board section for the row.
- `--top` — move the row to the top of its section.
- `--before REF` — move the row just above `REF`'s row (adopting its section).
- `--after REF` — move the row just below `REF`'s row (adopting its section).

### archive

`duty archive` — Move every `done` task in the current board and below into its
own board's `archive/`, dropping the row and fixing the count. Idempotent —
"nothing to archive" is a clean no-op.

```sh title="Archive finished work"
duty archive
```

Flags:
- `--in PATH` — archive from another board (`.` = root).

### delete task

`duty delete task <id>` — Remove an open task and its board row. Refuses a
`done` task without `--force` — that's `archive`'s job.

```sh title="Delete a scratch task"
duty delete task T-07 --force
```

Flags:
- `--force` — allow deleting a `done` task.

## Read

Reads never lock and never write. Each takes `--agent` for stable TSV — see
[Agent output](#agent-output) below.

### get tasks

`duty get tasks` — List open tasks from the current board down, straight from
the files: `id  status  title`, a relative age, a track prefix when not local,
and a `⚠ …` flag when the file and its board row disagree — `board says done`
when the statuses differ, `board says missing` for a file with no row, `no file`
for a row pointing nowhere, `unparsable` for a file whose frontmatter won't
parse. A bad file is listed, never an error. A task with unmet dependencies also
trails a dim `waits T-01,T-03` naming the ids still blocking it (a done or
archived dependency counts as met, the same rule `get next` walks). Rows come
out in board order — board order is priority order, the same order `get next`
walks — with a task whose file has no board row sorted by name into the default
section, still flagged.

```sh title="Only what's in progress"
duty get tasks --status in-progress
```

Flags:
- `--status S` — only this status.
- `--in PATH` — start from another board.
- `--agent` — TSV: `id`, `board-path`, `status`, `title`, `drift`, `updated`, `claimed-by`, `waits`.

### get task

`duty get task <id>` — One task's metadata, from its file (never its body): id,
title, status, track, blocked-by, gates `n/m`, age, and path — plus a
`claimed-by:` line when an `in-progress` task names its holder. Each `blocked-by`
id carries its status in parentheses (`T-01 (done)`, `T-03 (in-progress)`).
`--section` prints one section's body instead; `--body` prints the whole body.

```sh title="Inspect a task"
duty get task T-07
```

Flags:
- `--section NAME` — print only that section's body.
- `--body` — print the whole body below the frontmatter.
- `--agent` — one TSV record (fields under Agent output).

### get tracks

`duty get tracks` — One line per board, the root shown as `.`: path, title,
per-status counts of its own tasks (todo, in-progress, done, blocked, backlog),
and its archived count.

```sh title="Board-by-board counts"
duty get tracks
```

Flags:
- `--in PATH` — start from another board.
- `--agent` — TSV: `path`, `title`, `todo`, `in-progress`, `done`, `blocked`, `archived`, `backlog`.

### get next

`duty get next` — The first actionable task: board order first, then sub-tracks
depth-first — the first `todo` whose `blocked-by` are all done. A task whose
file has no board row still counts; it sorts into the default section, so a
rowless unblocked todo is reachable here. Nothing actionable means no output and
exit `0`. Output shape matches `get task`.

```sh title="Claim the next task"
duty get next --claim
```

Flags:
- `--claim` — atomically mark it `in-progress` and print it, so parallel agents each get a distinct task.
- `--as NAME` — with `--claim`, record NAME as the claimer (falls back to `$DUTY_AGENT`).
- `--in PATH` — start from another board.
- `--agent` — same fields as `get task`.

## Interface

### tui

`duty tui` — Launch the live board viewer: read-only, refreshing as files
change. See [TUI](/tui/).

```sh title="Watch the board"
duty tui
```

### watch

`duty watch` — Stream one line per task state change, live. It's the one
long-running command — every other `duty` runs and exits; `watch` blocks,
printing a line the moment a task changes and nothing until then. It prints
nothing on start (it reports state, not history), so pair it with one `get
tasks --agent` for the baseline. `Ctrl-C` exits `0`; if the tree disappears
underneath it, it exits non-zero with one line.

It watches the same files the TUI does, diffing consecutive scans and emitting
one line per changed field. Six kinds of change surface: `status`, `claimed-by`,
`created`, `deleted`, `moved` (a task changed board), and `gates` (a gate ticked
or added). Claiming a task shows two lines — the `status` and the `claimed-by`
both moved.

```sh title="React to every change (orchestrator)"
duty get tasks --agent      # baseline snapshot, once
duty watch --agent          # then stream changes
# 2026-07-16T12:53:57+02:00	status	T-02	status	todo	in-progress
# 2026-07-16T12:53:58+02:00	moved	T-07	board	.	backend
```

Flags:
- `--in PATH` — watch one board (and below) by its track path (`.` = root).
- `--agent` — TSV: `time`, `event`, `id`, `field`, `old`, `new` (`time` RFC3339).

### skill

`duty skill` — Print the duty agent skill to stdout: a short, token-lean brief
that teaches an agent the four-call loop and the working rules. `duty skill
install <harness>` writes it where a harness will load it:

- `claude` → `.claude/skills/duty/SKILL.md` (a Claude Code skill, frontmatter
  and all). `--user` installs it in your home directory instead of this repo.
- `codex` → a marker-delimited block in `AGENTS.md` at the current directory.
- `gemini` → the same block in `GEMINI.md`.

Run `duty skill install` with no harness on a terminal and it prompts you to
pick one. Both `skill` and `skill install` fetch the latest text from
`https://duty-cli.xyz/skill.md` and fall back silently to the copy baked into
the binary, so the skill can improve without a new release. `install` prints
`installed <target> skill → <path>` — the same line whether the content came
from the remote or the embedded fallback.

```sh title="Install the skill for Claude Code"
duty skill install claude
# installed claude skill → /abs/path/to/.claude/skills/duty/SKILL.md
```

Flags:
- `--user` — install for `claude` in your home directory, not this repo.
- `--force` — replace an existing install (refused otherwise).
- `--offline` — skip the network fetch and use the embedded copy.

## Board context (`--in`)

`create task`, `create track`, `get tasks`, `get tracks`, `get next`,
`archive`, and `watch` take `--in` to name a board by a root-relative slash path
(`.` = the root board): the tree root is found from cwd, then the board becomes
`<root>/<PATH>`. Id-addressed commands take no `--in` — ids resolve tree-wide.

## Agent output

Every read takes `--agent`: stable, token-lean TSV — one record per line, no
padding, no color, the field order part of the contract. New fields only ever
append, so parsers of the earlier fields keep working: `updated` (the file's
mtime, RFC3339) trails every record; `get task`/`get next` append `claimed-by`
after it, `get tasks` appends `claimed-by` then `waits`, and `get tracks`
appends `backlog` last. Lifecycle-mutating commands (`status`, `report`,
`gates`, `move`, `archive`, `delete`, `set`) stay quiet either way; the
scaffolding commands (`init`, `create task`, `create track`, `skill install`)
always print their one confirmation line — there's no other way to learn what
they made.

- `get tasks` — `id  board-path  status  title  drift  updated  claimed-by  waits` (drift empty, `board=<status>`, `no-row`, `no-file`, or `bad-file`; waits comma-joined blocked-by ids not yet met, empty when actionable).
- `get task` / `get next` — `id  track-path  status  title  gates-done  gates-total  blocked-by  path  updated  claimed-by` (blocked-by comma-joined; claimed-by empty unless an in-progress task names its holder; `get next` prints nothing when nothing's actionable).
- `get tracks` — `path  title  todo  in-progress  done  blocked  archived  backlog`.
- `watch` — `time  event  id  field  old  new` (`time` RFC3339; `event` one of `status`, `claimed-by`, `created`, `deleted`, `moved`, `gates`; one line per changed field, streamed until `Ctrl-C`).

## Cheat sheet

| Command | What it does |
|---|---|
| `duty init [title]` | scaffold a `duty/` tree here |
| `duty create task <title>` | add a task to the current board |
| `duty create track <name>` | add a nested track |
| `duty set <id> [section]` | replace section(s) from stdin |
| `duty status <id> <status>` | set status in file + board row |
| `duty report <id>` | append a report; `--status` flips status too |
| `duty gates <id>` | list gates; `add` / `check` / `uncheck` edit them |
| `duty move <id>` | move a task across tracks / sections, or reorder its row |
| `duty archive` | archive every done task, here and below |
| `duty delete task <id>` | remove an open task |
| `duty get tasks` | list open tasks from the files |
| `duty get task <id>` | one task's metadata |
| `duty get tracks` | per-board counts |
| `duty get next` | first actionable task; `--claim` takes it |
| `duty tui` | live board viewer |
| `duty watch` | stream one line per task state change (long-running) |
| `duty skill` | print the agent skill; `install <harness>` writes it |
