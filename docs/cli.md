# CLI

Verb → resource, kubectl-style: `create`, `get`, `delete` take a resource subcommand;
the agent hot path stays verb-only. Every mutating command maintains the sync
invariant (file + board in one shot) and serializes on a tree-wide write lock (see
internals.md). A recurring agent sequence that would take the lock twice earns a
one-shot form — `report --status`, `gates check --all`, variadic `gates add`: one
intent, one lock, one atomic write. Exit codes: `0` ok, `≠0` error with a one-line
stderr message; `2` for a missing or unknown command — typos suggest the closest
command within edit distance 2, still exit `2`. `duty --version` prints the build
version (`dev` unless set via `-ldflags "-X main.version=…"`). An id that resolves
nowhere hints: `unknown task id "T-99" — try 'duty get tasks'`. Help groups commands
by role (Author / Work / Read / Interface), opens with the lifecycle, and every
command has an `Example`.

| Command | Behavior |
|---|---|
| `duty init [title]` | Bootstrap: create `duty/` in cwd with skeleton `BOARD.md` (H1 = title, default `Board`), `README.md`, `archive/`. Refuse if already inside a tree. |
| `duty create task <title> [--slug S] [--blocked-by ID…] [--section NAME] [--in PATH] [--body]` | Create in the **current board** (or the `--in` board). Validate every `--blocked-by` id exists (anywhere in the tree). Next NN scans the whole tree. Write the task file (frontmatter + generated H1 + body) and append the board row (`todo`) to the section's table — default `Open tasks`, created if absent — under one lock. Body: the section skeleton by default, or with **`--body`** the entire markdown body (everything below the H1) read from stdin — `## ` sections verbatim, gates as `- [ ]` checkboxes under `## Gates`; `--body` refuses empty stdin and requires the body to open at a `## ` heading. Print the created path. |
| `duty create track <name> [--title T] [--in PATH]` | Create track `<name>/` (validated `[a-z0-9-]+`) under the current board (or the `--in` board): skeleton `BOARD.md` (H1 = title, default: the name) + `archive/`. Append its bullet to the parent's `## Boards` (created if absent). Refuse if the folder exists. |
| `duty status <id> <status> [--force]` | Rewrite the frontmatter `status:` line + the row's status cell. Reject unknown statuses. Setting `in-progress` on a task already `in-progress` is refused (`T-x is already in-progress — someone claimed it; use --force to take it over`) unless `--force` — the one guarded transition, so a live claim is never silently stolen; every other transition stays free (see tasks.md). |
| `duty move <id> [--track PATH] [--section NAME]` | At least one flag. `--track` moves the task to another track: `PATH` is relative to the tree root (`.` = root board); rename the file into the target folder (same filename — ids don't encode tracks), drop the source row (prune), append to the target's `--section` (default `Open tasks`), status preserved. `--section` alone moves the row under `## <section>` within its own board (created if absent, inserted above the footer); prune any section left empty. |
| `duty report <id> [--status S] [--force]` | Append stdin under `## Report` (heading created once, content accumulates). With `--status S` also flip the status (frontmatter + board row) in the *same* locked write — the atomic done/blocked lifecycle endings; both edits are computed before either file is written, so any error lands neither. `--status in-progress` obeys the claim guard (`--force` to take over). Refuse empty stdin. |
| `duty set <id> [section]` | With a `section` argument, replace that one section's body from stdin, line-surgically (heading line and every byte outside the section survive); the name matches the `## <name>` heading case-insensitively, a missing section is created before `## Report` (or at end of file). With **no argument**, stdin is one or more `## Section` blocks and each is replaced (created if missing) in payload order, all under one lock in one file write — the bulk form must open at a `## ` heading. Refuse empty stdin either way. |
| `duty gates <id> [list]` | List the task's gates, 1-based (`1 [x] build passes`); `--agent` emits `index<TAB>done<TAB>text` TSV. `gates add <id> <text>...` appends a `- [ ]` gate per text, in order, in one write (creating `## Gates` if absent); `gates check <id> <n>` / `gates uncheck <id> <n>` flip the n-th checkbox surgically (erroring when `n` is out of range), or `--all` flips every gate in one write. |
| `duty archive [--in PATH]` | For every open task with `status: done`, in the current board (or the `--in` board) and every board below it: rename → its own board's `archive/`, drop its row, prune empty sections, rewrite that board's footer count. Idempotent; "nothing to archive" is a clean no-op. |
| `duty delete task <id> [--force]` | Refuse on `done` without `--force` (that's `archive`'s job). Remove the file, drop the row, prune. |
| `duty get tasks [--status S] [--in PATH]` | Recursive from the current board (or the `--in` board). One line per open task **from the files**: `id  status  title`, prefixed with the track path when not local (`backend/ T-12 …`), closed by a dim relative-age column (`2h ago` / `3d ago`; "just now" under a minute; the absolute date past seven days). If the board row's status disagrees (or the row is missing), append a `⚠ board says …` drift flag. `list` survives as a hidden alias. |
| `duty get task <id> [--section NAME] [--body]` | One task's metadata **from its file** — never its body (the path is printed; readers `cat` it). Resolves the id anywhere in the tree. Human: aligned `key: value` lines (id, title, status, track, blocked-by, gates `n/m`, updated (relative age), path). `--section NAME` instead prints that one section's body (exit 1 `no section "x" in T-05` when absent); `--body` prints the entire body below the frontmatter, verbatim (read-only, no lock). `--body`, `--section`, and `--agent` are mutually exclusive. |
| `duty get tracks [--in PATH]` | One line per board — the starting board included as `.` — recursive from the current board (or the `--in` board): path, title, per-status counts of its **own** (directly-filed) tasks, and its archived count. |
| `duty get next [--claim] [--in PATH]` | The first **actionable** task: walk the current board's (or the `--in` board's) rows in board order (build order is priority), then sub-tracks depth-first in scan order; return the first `todo` whose `blocked-by` are all `done` (an archived dependency counts as done). Output shape = `get task`. **Nothing actionable → no output, exit 0.** With **`--claim`** it atomically marks that task `in-progress` (file + board row) under the tree-wide lock before printing it — the printed status is the truthful `in-progress`, the `--agent` shape unchanged — so parallel agents each get a distinct task and losers of the race transparently receive the following one; a claim with nothing to do stays a pure read (no lock-file side effect). |
| `duty tui` | Launch the live board viewer (see tui.md). |

**Board context (`--in PATH`).** `create task`, `create track`, `get tasks`,
`get tracks`, `get next`, and `archive` take a long-only `--in` naming the board by a
**root-relative** slash path (`.` = root board): the tree root is still resolved from
cwd, then the current board becomes `<root>/<PATH>`. The path must name an existing
board, else `unknown track "api/auth"` — the one validator and error shape
`move --track` uses. Id-addressed commands take no `--in`: ids resolve tree-wide.

**One-shot authoring (the agent write path).** `create task --body` and bulk `set`
author a whole task in a single call each. The body is fed as **markdown, not JSON**:
markdown-in splices byte-for-byte, JSON would double-encode and re-render — the drift
the line-surgical writers exist to avoid.

**Agent output.** Reading commands accept `--agent` (long-only): stable, token-lean
TSV — one record per line, no padding, no color; the field order is part of the
contract. Mutating commands stay quiet either way.

- `get tasks --agent` — `id<TAB>board-path<TAB>status<TAB>title<TAB>drift<TAB>updated`
  (drift empty, or `board=<status>`, or `no-row`).
- `get task --agent` / `get next --agent` — one record
  `id<TAB>track-path<TAB>status<TAB>title<TAB>gates-done<TAB>gates-total<TAB>blocked-by<TAB>path<TAB>updated`
  (blocked-by comma-joined, empty when none); `get next` prints nothing when nothing
  is actionable.
- `get tracks --agent` — one record per board
  `path<TAB>title<TAB>todo<TAB>in-progress<TAB>done<TAB>blocked<TAB>archived`.

`updated` is **trailing** so parsers of earlier positional fields keep working; it is
the file's mtime (RFC3339), not a frontmatter timestamp — no task file gains a field.

**Behavioral invariants:**
- **Lossless round-trip:** create → status → report → move to a section → move to
  another track → move back → delete → archive on a scratch task leaves the `duty/`
  tree byte-identical (hash before/after) — every writer preserves what it doesn't own.
- **Board edits are line-surgical:** touch only the target line/cell, write the rest
  back verbatim — never re-render the board from a model.
- Section pruning never removes the default section, and `get tasks` reads files as
  truth and only *reports* drift — never auto-heals.
