# Spec — `duty`: a file-based task system with a sync-keeping CLI and a live TUI

Markdown task files + a board index + one Go binary (`duty`). Boards nest: any folder
with a `BOARD.md` is a board, recursively. No database, no daemon, no git dependency.

## 1. Principles

- **One file = one task** — plain text, small enough for one worker for one sitting.
- **Boards nest by convention:** a directory with a `BOARD.md` is a board, recursively;
  tooling discovers the tree by scanning, never by registration.
- **The task file is the truth; the board is a projection** (ordering + grouping) —
  on disagreement the file wins and tooling *flags* the drift.
- **The sync invariant:** every state change writes frontmatter AND board row in one command.
- **The TUI is a viewer, not a second writer** — mutations go through the CLI or `$EDITOR`.
- **Tasks carry directives, never code** — the file accumulates *reports* about the work.
- **Agent-first CLI:** one-shot, quiet, non-zero exit + one-line stderr on any problem.

## 2. Layout

```
duty/
  duty.toml                — project config (optional; also marks the tree root)
  README.md                — the convention (human + agent contract, one page)
  BOARD.md                 — this board's index: one row per open task
  T-NN-<slug>.md           — one open task each
  archive/                 — this board's completed tasks, moved here verbatim
  backend/                 — a track: same shape, all the way down
    BOARD.md
    T-NN-<slug>.md
    archive/
```

Every board is self-contained: its own tasks, its own `archive/`. A track is a folder;
its board defines its state. The **current board** is the nearest ancestor of cwd
containing a `BOARD.md` (git-style walk-up; fallback `./duty/` from outside the tree).
Creating commands target the current board; id-taking commands resolve tree-wide.

## 3. The task file

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
  legal; discipline lives in the lifecycle contract (§5), not a state machine.

## 4. The board

`BOARD.md` is the only home for **ordering** (files can't express priority) and the
one page showing a board's state at a glance.

```markdown
# Board

Convention: [README.md](README.md). Workers update their row's status via the CLI.
Order top-to-bottom is the intended build order.

## Boards

- [backend/](backend/BOARD.md) — Backend

## Open tasks

| Task | Title | Status |
|------|-------|--------|
| [T-01](T-01-short-title.md) | Short imperative title | todo |

Completed tasks (12) archived: [archive/](archive/).
```

- The H1 is the board's **title** (root default `Board`; tracks get theirs from
  `create track`) — the truth for the TUI breadcrumb and track rows. The Task cell
  links bare id → filename; row lookup keys on the `(filename)` substring.
- **3 columns, no column creep.** Status is the one denormalization, kept honest by
  the sync invariant; everything else is read from the files.
- **`## Boards`** bullets (`- [name/](name/BOARD.md) — Title`) are for humans:
  `create track` appends one, tooling **never reads them**. Omitted when no tracks.
- **Sections** are `## <Name>` headers, each with its own table. `## Open tasks` is
  the default and always exists; others are created on demand and **pruned** empty.
  New sections insert *above* the footer `Completed tasks (N) archived:
  [archive/](archive/).`, whose count `archive` regex-rewrites.

## 5. Lifecycle (the worker's contract)

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

## 6. The CLI

Verb → resource, kubectl-style: `create`, `get`, `delete` take a resource subcommand;
the agent hot path stays verb-only. Every mutating command maintains the sync
invariant (file + board in one shot) and serializes on a tree-wide write lock (§10). A
recurring agent sequence that would take the lock twice earns a one-shot form —
`report --status`, `gates check --all`, variadic `gates add`: one intent, one lock,
one atomic write. Exit codes: `0` ok, `≠0` error with a one-line stderr message; `2`
for a missing or unknown command — typos suggest the closest command within edit
distance 2, still exit `2`. `duty --version` prints the build version (`dev` unless
set via `-ldflags "-X main.version=…"`). An id that resolves nowhere hints:
`unknown task id "T-99" — try 'duty get tasks'`. Help groups commands by role (Author /
Work / Read / Interface), opens with the lifecycle, and every command has an `Example`.

| Command | Behavior |
|---|---|
| `duty init [title]` | Bootstrap: create `duty/` in cwd with skeleton `BOARD.md` (H1 = title, default `Board`), `README.md`, `archive/`. Refuse if already inside a tree. |
| `duty create task <title> [--slug S] [--blocked-by ID…] [--section NAME] [--in PATH] [--body]` | Create in the **current board** (or the `--in` board). Validate every `--blocked-by` id exists (anywhere in the tree). Next NN scans the whole tree. Write the task file (frontmatter + generated H1 + body) and append the board row (`todo`) to the section's table — default `Open tasks`, created if absent — under one lock. Body: the section skeleton by default, or with **`--body`** the entire markdown body (everything below the H1) read from stdin — `## ` sections verbatim, gates as `- [ ]` checkboxes under `## Gates`; `--body` refuses empty stdin and requires the body to open at a `## ` heading. Print the created path. |
| `duty create track <name> [--title T] [--in PATH]` | Create track `<name>/` (validated `[a-z0-9-]+`) under the current board (or the `--in` board): skeleton `BOARD.md` (H1 = title, default: the name) + `archive/`. Append its bullet to the parent's `## Boards` (created if absent). Refuse if the folder exists. |
| `duty status <id> <status> [--force]` | Rewrite the frontmatter `status:` line + the row's status cell. Reject unknown statuses. Setting `in-progress` on a task already `in-progress` is refused (`T-x is already in-progress — someone claimed it; use --force to take it over`) unless `--force` — the one guarded transition, so a live claim is never silently stolen; every other transition stays free (§3). |
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
| `duty tui` | Launch the live board viewer (§8). |

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

**Behavioral invariants (test these):**
- **Lossless round-trip:** create → status → report → move to a section → move to
  another track → move back → delete → archive on a scratch task leaves the `duty/`
  tree byte-identical (hash before/after) — every writer preserves what it doesn't own.
- **Board edits are line-surgical:** touch only the target line/cell, write the rest
  back verbatim — never re-render the board from a model.
- Section pruning never removes the default section, and `get tasks` reads files as
  truth and only *reports* drift — never auto-heals.

## 7. Configuration

TOML, read-only, merged over built-in defaults: **user**
(`os.UserConfigDir()/duty/config.toml`) < **project** (`duty.toml` next to the root
`BOARD.md`; its presence also marks the tree root explicitly — otherwise the walk-up
stops at the topmost `BOARD.md`). Missing files are fine. Only the root `duty.toml` is
read; one inside a track is an error (a second root).

```toml
editor = "nvim"        # falls back to $EDITOR, then vi

[tui]
theme = "auto"         # auto | dark | light — background mode, not colors

[tui.palette]          # optional per-slot color overrides; unset = the §8 default
accent = "#e1ebaf"     # a bare string sets both the light and dark channel
todo = { light = "#8a6d00", dark = "#af874b" }   # or set each channel apart
```

`[tui.palette]` recolors the TUI's semantic slots — `accent`, `dim`, `todo`,
`in-progress`, `done`, `blocked` — over the frozen §8 default; an omitted slot (or,
in the table form, an omitted channel) keeps its default, so the zero config renders
the default palette byte-for-byte. Each value is a `#rrggbb` hex triplet or an ansi
index `0-255`; a malformed value is an error naming the key
(`tui.palette.todo.dark`). The palette is a distinct table from the `theme` mode
selector above (TOML forbids one key being both a string and a table). Status colors
ink the word directly and their dark hue fills that status's distribution bar, so one
slot drives both.

Keys get added when a hardcoded value hurts, not before. Config tunes presentation
only — statuses, naming, and board structure are the convention, never settings.

## 8. The TUI

`duty tui` — a read-only live board. **Per frame:** scan the tree for boards, parse
each `BOARD.md` for sections + row order, parse every task's frontmatter and count its
gates. Files win; a board/file mismatch renders a `⚠` badge on the row.

**Layout — browse full-width, preview on open** (all styling lipgloss). Startup is
instant: no markdown renderer is built and no terminal query fires until a task is *opened*.

- **Header:** breadcrumb of the track path — each board's H1 title (§4), each segment
  a clickable zone jumping to that ancestor — plus the **subtree**'s per-status counts
  in status colors and a one-line status-distribution bar.
- **Left panel** (full width browsing; ~38%, min 30 cols, with a preview open): a
  `bubbles/list`, sub-tracks first under a non-selectable **"Tracks" header** (skipped
  by navigation, hidden while filtering) — name and title left, then a **right-aligned
  fixed-width status-distribution bar** of its subtree flush at the line end with a
  dim total, the bar column at the same x on every row
  (`backend/  Title      ▰▰▰▰▰▰▱▱  7`): proportional colored segments, every non-zero
  status ≥1 cell, a dim `empty` when taskless; the title ellipsis-truncates first, the
  bar is never dropped. Then tasks under their section headers: id, title, colored
  status word, gates `2/3`, dim relative age, drift badge. The **age column is always
  shown** (`t` toggles); the **gate column hides below 100 columns**. Rows within a
  section are **status-grouped for display by default** — in-progress, todo, blocked,
  done, unknown last — a *stable* sort, board build order as tiebreak, presentation
  only; `s` toggles raw board order (session-only, no config key). `/` opens the fuzzy
  filter — match rank orders rows while active, the status sort steps aside. Empty
  boards show a centered dim hint; a no-match filter shows the list's no-items line.
- **Colors.** Dark theme (frozen): status words carry the raw duty palette as
  foreground — `todo` bronze, `in-progress` peach, `done` olive, `blocked` red —
  accents cream. Light theme: the raw hues are too pale on white, so status words are
  **flat AA-darkened inks** (no chips, no backgrounds): accent/ink navy `#1f3a5f`
  (11.5:1), in-progress blue `#3a6ea5` (5.3:1), todo amber `#8a6d00` (4.9:1), done
  olive `#6f7d27` (4.5:1), blocked red (5.4:1), dim grey (5.25:1), body black. Bars
  fill with the raw hues on both themes. The **selected row is bold across the whole
  line**, both themes. This palette is the default; every slot is overridable from
  `[tui.palette]` (§7).
- **Right panel — on open only:** `enter`/double-click on a task opens the split — the
  file rendered by glamour in a viewport, focus on the preview; `esc` closes. `enter`
  on a track descends; with a preview open it shows the track's summary card (title,
  per-status counts, bar, sections with row counts, subtree drift count). A task
  preview is topped by a pinned header — `id · status (colored) · gates n/m · track
  title · age` (age dim, trailing last so narrow truncation drops it before the
  `blocked-by` ids and drift flag). The preview reads from the rows' scan snapshot
  (zero extra I/O); one glamour renderer, built lazily on first open, rebuilt only on
  width change; content re-renders on re-scan.
- **Footer:** key-hint bar (`?` toggles the full grid). **Responsive:** below ~80
  columns an open preview takes over the body full-screen.

**Keys:** `j/k` move, `enter` open/descend, `esc` back (close preview / clear filter /
up one track), `tab` panel focus, `/` filter, `e` open in `$EDITOR` (suspend, resume),
`t` age column (default on), `s` sort toggle (default grouped), `r` manual re-scan,
`?` help, `q` quit. **Mouse:** panels, rows, and breadcrumb segments are hit-zones —
click selects/focuses/jumps, double-click opens/descends, the wheel scrolls the
hovered panel; preview scrolling is spring-smoothed, never slower than the keyboard.

**Live refresh:** fsnotify watches every directory (per-directory, not recursive; a
directory event re-walks so new tracks appear live). Debounce ~100 ms, then **re-scan
everything** — a full re-read beats any cache and keeps the TUI stateless. No polling,
no IPC.

## 9. Implementation notes (Go)

- **One module, ports-and-adapters** — layout and code rules live in `CLAUDE.md`.
  Cobra subcommands, thin (parse → app service → format), cobra's own printing
  silenced: quiet on success, one lowercase stderr line + non-zero exit on error.
- **Deps:** `spf13/cobra`, `BurntSushi/toml`, `gopkg.in/yaml.v3` (frontmatter *read*),
  `fsnotify`, `gofrs/flock`, and the Charm stack — `bubbletea`, `bubbles` (never
  hand-roll a component bubbles ships), `lipgloss` (all styling; no raw ANSI),
  `glamour`, `bubblezone`, `harmonica`, `ntcharts`.
- **Writes are targeted line edits**, never YAML re-serialization; **and atomic** —
  temp file in the same directory, rename over the target — no half-written reads.
- Frontmatter parse: `\A---\n(.*?)\n---\n` + `yaml.Unmarshal`. Status write:
  `(?m)^status: \S+`, first match only.
- Row find: the line containing `(<filename>)` that starts with `|`. Status cell:
  `strings.Split(row, "|")`, replace `cells[len(cells)-2]`, rejoin — preserves spacing.
- Task-id → file: walk from the tree root matching `<id>-*.md`; unknown ids hint
  `try 'duty get tasks'`, an archived match notes archived tasks are read-only. Global
  NN uniqueness (§3) guarantees at most one match.
- Board discovery: walk collecting directories containing `BOARD.md`, skipping
  `archive/`. Reused by reads, archive, the TUI scan, and NN numbering.
- Gate count: `- [x]` vs `- [ ]` lines under `## Gates` (until the next `^## `).
- Companion agent doc `duty/README.md` (one page): command table, lifecycle→command
  mapping, what stays the worker's judgment. Generated skeletons (task file, board,
  `duty/README.md`) are `go:embed`ed `.md.tmpl` templates (`text/template`).

## 10. Deliberately not built (YAGNI — add only when it hurts)

- **No state machine** on transitions — a flat setter + a written contract.
- **No due dates, priorities, assignees, labels, timestamps** — board order *is* the
  priority.
- **No dependency enforcement** beyond existence-check at create — `blocked-by` is
  advisory; the worker contract enforces it.
- **No board regeneration command** — boards are edited surgically and drift is
  *surfaced*, never rebuilt.
- **No editing archived tasks** — the CLI refuses to resolve ids there.
- **No TUI mutations** — viewer only: no status-cycling keys, no in-TUI forms;
  `$EDITOR` + the CLI are the write path and the watcher shows them instantly.
- **No board delete/move commands** — tasks move; boards don't. Delete a board by
  deleting the folder and fixing the parent's `## Boards` bullet by hand (cosmetic).
- **Tree-wide write lock, nothing finer** — every mutating command (`create`,
  `status`, `report`, `set`, `gates`, `move`, `archive`, `delete`; not `init`, not
  reads) holds one advisory `flock` on `<root>/.duty.lock` (gofrs/flock; created on
  demand, gitignored, never committed) for its whole duration; `get next --claim`
  re-scans and marks the task `in-progress` under that same lock, so parallel claims
  each hand back a distinct task. A writer that can't take the lock within 5s fails
  with `tree is locked` rather than racing on a `BOARD.md`. Deliberately coarse: no
  per-task granularity, no lease / heartbeat / stale-claim recovery — a crashed agent
  leaves its task `in-progress`; `duty status <id> in-progress --force` is the manual
  takeover. Reads never lock.
- **No stored rollups** — track counts are computed live from files, written nowhere.
- **No semantic config** — `duty.toml` tunes presentation (§7). A config key that
  changes file formats is a bug.
