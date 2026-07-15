# CLI

`duty` is verb-first, kubectl-style: `create`, `get`, and `delete` take a
resource (`task` / `track`); every other command is a bare verb. Whatever
writes keeps the task file and its board row in sync in one atomic step, under a
tree-wide lock — so a swarm of agents never corrupts a board.

It's quiet when it works. An error is one lowercase line on stderr and a
non-zero exit; a missing or unknown command exits `2` and suggests the closest
match. `duty --version` prints the build version.

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
tree.

```sh title="Start a board"
duty init "Q3 roadmap"
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
name must be `[a-z0-9-]+`.

```sh title="A backend track"
duty create track backend --title "Backend work"
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
`done`, or `blocked` — in both its file and its board row. Every transition is
free except claiming a task already `in-progress`, refused so a live claim is
never silently stolen.

```sh title="Start a task"
duty status T-07 in-progress
```

Flags:
- `--force` — take over a task already `in-progress`.

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
and a `⚠ board says …` flag when the board row disagrees.

```sh title="Only what's in progress"
duty get tasks --status in-progress
```

Flags:
- `--status S` — only this status.
- `--in PATH` — start from another board.
- `--agent` — TSV: `id`, `board-path`, `status`, `title`, `drift`, `updated`.

### get task

`duty get task <id>` — One task's metadata, from its file (never its body): id,
title, status, track, blocked-by, gates `n/m`, age, and path. `--section`
prints one section's body instead; `--body` prints the whole body.

```sh title="Inspect a task"
duty get task T-07
```

Flags:
- `--section NAME` — print only that section's body.
- `--body` — print the whole body below the frontmatter.
- `--agent` — one TSV record (fields under Agent output).

### get tracks

`duty get tracks` — One line per board, the root shown as `.`: path, title,
per-status counts of its own tasks, and its archived count.

```sh title="Board-by-board counts"
duty get tracks
```

Flags:
- `--in PATH` — start from another board.
- `--agent` — TSV: `path`, `title`, `todo`, `in-progress`, `done`, `blocked`, `archived`.

### get next

`duty get next` — The first actionable task: board order first, then sub-tracks
depth-first — the first `todo` whose `blocked-by` are all done. Nothing
actionable means no output and exit `0`. Output shape matches `get task`.

```sh title="Claim the next task"
duty get next --claim
```

Flags:
- `--claim` — atomically mark it `in-progress` and print it, so parallel agents each get a distinct task.
- `--in PATH` — start from another board.
- `--agent` — same fields as `get task`.

## Interface

### tui

`duty tui` — Launch the live board viewer: read-only, refreshing as files
change. See [TUI](/tui/).

```sh title="Watch the board"
duty tui
```

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
the binary, so the skill can improve without a new release.

```sh title="Install the skill for Claude Code"
duty skill install claude
```

Flags:
- `--user` — install for `claude` in your home directory, not this repo.
- `--force` — replace an existing install (refused otherwise).
- `--offline` — skip the network fetch and use the embedded copy.

## Board context (`--in`)

`create task`, `create track`, `get tasks`, `get tracks`, `get next`, and
`archive` take `--in` to name a board by a root-relative slash path (`.` = the
root board): the tree root is found from cwd, then the board becomes
`<root>/<PATH>`. Id-addressed commands take no `--in` — ids resolve tree-wide.

## Agent output

Every read takes `--agent`: stable, token-lean TSV — one record per line, no
padding, no color, the field order part of the contract. `updated` is always
the trailing field (the file's mtime, RFC3339) so parsers of the earlier fields
keep working. Mutating commands stay quiet either way, except `create task`,
which always prints its one `<id><TAB><path>` line — there's no other way to
learn the id it picked.

- `get tasks` — `id  board-path  status  title  drift  updated` (drift empty, `board=<status>`, or `no-row`).
- `get task` / `get next` — `id  track-path  status  title  gates-done  gates-total  blocked-by  path  updated` (blocked-by comma-joined; `get next` prints nothing when nothing's actionable).
- `get tracks` — `path  title  todo  in-progress  done  blocked  archived`.

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
| `duty skill` | print the agent skill; `install <harness>` writes it |
