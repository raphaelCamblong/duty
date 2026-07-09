# Spec — `duty`: a file-based task system with a sync-keeping CLI and a live TUI

Markdown task files + a board index + one Go binary (`duty`). Boards nest: any folder
with a `BOARD.md` is a board, recursively. Everything here is buildable from scratch in
a day. No database, no daemon, no git dependency — just files the CLI keeps in sync and
the TUI watches.

## 1. Principles (the why)

- **One file = one task.** A task is a markdown file small enough to hand to one worker
  (human or agent) for one sitting. Plain text, greppable, diffable, no database.
- **Boards nest by convention, not registration.** Any directory containing a `BOARD.md`
  is a board; a subdirectory with its own `BOARD.md` is a sub-board, recursively. Tooling
  discovers the tree by scanning — the filesystem is the registry, so structure can't drift.
- **The task file is the source of truth; the board is a projection.** `BOARD.md` exists
  for ordering and grouping, but when they disagree, the file wins — and the tooling
  *detects* the disagreement (drift flagging) rather than silently trusting either side.
- **The sync invariant.** Every state change writes the task file's frontmatter AND its
  board row in one command. Hand-editing one side is how drift happens; the CLI exists to
  make the synced write the path of least resistance.
- **The TUI is a viewer, not a second writer.** It renders from the files (truth), laid
  out by the board (order + sections), and refreshes when the folder changes. Mutations
  go through the CLI or `$EDITOR` — never through TUI-internal state.
- **Tasks carry directives, never code.** A task states goal, scope, and acceptance
  gates. The work product lives in the repo; the task file accumulates *reports* about it.
- **Agent-first CLI.** Commands are one-shot, quiet, and exit non-zero with a one-line
  stderr message on any problem — so an LLM agent (or a script) can drive the whole
  lifecycle without parsing prose.

## 2. Layout

```
duty/
  duty.toml                — project config (optional; also marks the tree root)
  README.md                — the convention (human + agent contract, one page)
  BOARD.md                 — this board's index: one row per open task
  T-NN-<slug>.md           — one open task each
  archive/                 — this board's completed tasks, moved here verbatim
  backend/                 — a sub-board: same shape, all the way down
    BOARD.md
    T-NN-<slug>.md
    archive/
```

Every board is self-contained: its own tasks, its own `archive/`. The tree is discovered
by walking directories for `BOARD.md` files — no manifest, no registration. There are
**no tracks**: the grouping a track prefix used to carry lives in the folder structure —
one area of work = one sub-board.

The binary is `duty` (single Go module, `package main`). The **current board** is the
nearest ancestor of cwd containing a `BOARD.md` (git-style walk-up; from outside the
tree, falls back to `./duty/`). Commands that create things target the current board;
commands that take an id resolve it anywhere in the tree.

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

**Format choices (all three consumers — TUI, editor, agent — drive them):**
- **`##` headings, not bold labels.** Sections split mechanically on `^## `, render as
  real sections in any markdown renderer, and give agents unambiguous structure.
- **Gates are a `- [ ]` checklist.** Workers tick boxes as gates pass; tooling counts
  `[x]` vs `[ ]` for a progress readout (`2/3` in the TUI) without parsing prose.
- **Frontmatter is the only machine-owned region.** Everything below it is freeform
  markdown the tooling appends to but never rewrites.

**Naming:** `T-NN-<slug>.md`. `NN` is the next integer across the **entire tree** — all
boards, open AND archived (an archived T-07 anywhere blocks reuse) — zero-padded to two
digits, growing naturally past 99. Tree-wide uniqueness is what lets every command take a
bare id with no board path; which board a task belongs to is simply which folder it sits
in, so `move` never renames. `slug` is derived from the title (lowercase,
non-alphanumerics → `-`, collapsed, ≤40 chars).

**Statuses:** `todo | in-progress | done | blocked`. Flat setter — any transition is legal;
the discipline lives in the lifecycle contract (§5), not a state machine.

## 4. The board

Kept deliberately even though a pure folder scan could replace it: `BOARD.md` is the only
home for **ordering** (files can't express priority) and the one page — human or agent —
that shows a board's major state at a glance.

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

- The H1 is the board's **title** (root default `Board`; sub-boards get theirs from
  `duty board`). It's the truth for the TUI breadcrumb and sub-board rows; the parent's
  `## Boards` bullet text is cosmetic.
- The Task cell is a link: text = bare id, href = the filename. Row lookup keys on the
  `(filename)` substring — unambiguous and section-agnostic.
- The board stays **3 columns**. Status is duplicated here (the one denormalization, kept
  honest by the sync invariant); everything else — `blocked-by`, gate progress — is read
  from the files by whoever needs it. No column creep.
- **`## Boards`** lists sub-boards, one bullet each (`- [name/](name/BOARD.md) — Title`).
  It exists purely for humans browsing the markdown: `duty board` appends the bullet as a
  courtesy, but tooling **never reads it** — sub-board discovery is by scan, so a stale
  bullet is cosmetic, never a correctness problem. Omitted when there are no sub-boards.
- **Sections** are `## <Name>` headers, each followed by its own table. `## Open tasks`
  is the default and always exists; other sections are created on demand and **pruned
  automatically** when their last row leaves (empty sections are noise).
- The footer line `Completed tasks (N) archived: [archive/](archive/).` is maintained by
  the archive command (regex-rewrites the count). New sections insert *above* it.

## 5. Lifecycle (the worker's contract)

1. **Start** → `duty status <id> in-progress`.
2. **Blocked** (missing input, failed dependency, a decision the task doesn't pre-make) →
   `duty status <id> blocked` + pipe a report naming *exactly* what's missing, then stop.
   Never guess past a blocker.
3. **Working** → tick gate checkboxes in the task file as they pass.
4. **Done** (all gates ticked) → `duty status <id> done` + pipe a report: files changed,
   gate output tails, deviations (with why), follow-ups deliberately left.
5. Reports **accumulate** in the file — appended, never overwritten.
6. Respect `blocked-by`: don't start a task whose dependencies aren't `done`.
7. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.

## 6. The CLI

Subcommand-style. Every mutating command maintains the sync invariant (file + board in
one shot). Exit codes: `0` ok, `≠0` error with a one-line stderr message.

| Command | Behavior |
|---|---|
| `duty init [title]` | Bootstrap: create `duty/` in cwd with skeleton `BOARD.md` (H1 = title, default `Board`), `README.md`, `archive/`. Refuse if already inside a tree. |
| `duty create <title> [--slug S] [--blocked-by ID…] [--section NAME]` | Create in the **current board**. Validate that every `--blocked-by` id exists (anywhere in the tree). Next NN scans the whole tree. Write the template file (frontmatter + section skeleton). Append the board row (`todo`) to the section's table — default `Open tasks`, section created if absent. Print the created path. |
| `duty board <name> [--title T]` | Create sub-board `<name>/` (validated `[a-z0-9-]+`) under the current board: skeleton `BOARD.md` (H1 = title, default: the name) + `archive/`. Append its bullet to the parent's `## Boards` section (created if absent). Refuse if the folder already exists. |
| `duty status <id> <status>` | Rewrite the frontmatter `status:` line + the row's status cell. Reject unknown statuses. |
| `duty link <id> <section>` | Move the row under `## <section>` (created if absent, inserted above the footer); prune any section left empty. |
| `duty move <id> <board-path>` | Move a task to another board. `board-path` is relative to the tree root (`.` = root board). Rename the file into the target folder (same filename — ids don't encode boards), drop the source row (prune), append to the target's `Open tasks` (or `--section`), status preserved. |
| `duty report <id>` | Append stdin under `## Report` (heading created once, content accumulates). Refuse empty stdin. |
| `duty archive` | For every open task with `status: done`, in the current board and every board below it: `os.Rename` → its own board's `archive/`, drop its row, prune empty sections, rewrite that board's footer count. Idempotent; "nothing to archive" is a clean no-op. |
| `duty delete <id> [--force]` | Refuse on `done` without `--force` (that's `archive`'s job). Remove the file, drop the row, prune. |
| `duty list [--status S]` | Recursive from the current board. One line per open task **from the files**: `id  status  title`, prefixed with the sub-board path when not local (`backend/ T-12 …`). If the board row's status disagrees (or the row is missing), append a `⚠ board says …` drift flag. |
| `duty tui` | Launch the live board viewer (§8). |

**Agent output.** Reading commands accept `--agent` (long-only, no shorthand): stable,
token-lean TSV — one record per line, tab-separated fields, no alignment padding, no
color, no badges. `duty list --agent` emits `id<TAB>board-path<TAB>status<TAB>title<TAB>drift`
(drift empty, or `board=<status>`, or `no-row`). TSV, not JSON: fewer tokens, trivially
`cut`/`awk`-able, and the field order is part of the contract. Mutating commands stay
quiet either way.

**Behavioral invariants (test these):**
- **Lossless round-trip:** create → status → report → link → move → move back → delete →
  archive on a scratch task leaves the `duty/` tree byte-identical (hash it before and after). This
  is the master acceptance test — it proves every writer preserves everything it doesn't own.
- **Board edits are line-surgical:** read lines, touch only the target line/cell, write
  back. Never re-render the whole board from a model (that's how hand-written prose,
  banners, and ordering get destroyed).
- Section pruning never removes the default section.
- `list` reads files as truth and only *reports* drift — it never auto-heals.

## 7. Configuration

TOML, read-only, two locations merged over built-in defaults:

1. **User:** `os.UserConfigDir()/duty/config.toml` (`~/.config/duty/config.toml` on Linux).
2. **Project:** `duty.toml` next to the root `BOARD.md`. Its presence also marks the
   tree root explicitly (the walk-up otherwise stops at the topmost `BOARD.md`).

Project overrides user overrides defaults. Missing files are fine — everything works
with zero config. Only the root `duty.toml` is read; one inside a sub-board is an error
(it would declare a second root).

```toml
editor = "nvim"        # falls back to $EDITOR, then vi

[tui]
theme = "auto"         # auto | dark | light
```

Keys get added when a hardcoded value hurts, not before. Config tunes presentation
only — statuses, naming, and board structure are the convention, never settings.

## 8. The TUI

`duty tui` — a read-only live board.

**Data model per frame:** scan the tree for boards; parse each `BOARD.md` for sections +
row order; parse every task file's frontmatter for status/title/blocked-by and count its
gate checkboxes. Files win; a board/file mismatch renders as a `⚠` badge on the row — the
TUI is the always-on drift surfacer.

**Layout:**
- **Board view** (one board at a time, breadcrumb of the board path on top): sub-boards
  first, one line each with live counts computed from files (`backend/  3/7 done`); then
  sections as headers, one line per task: id, title, colored status (`todo` dim,
  `in-progress` yellow, `blocked` red, `done` green), gate progress `2/3`, drift badge
  if any.
- **Detail view** (on a task): the task file rendered as markdown, full screen,
  scrollable (`bubbles/viewport`).
- **Board header:** breadcrumb + a one-line status-distribution bar (ntcharts) so a
  board's health reads at a glance. Breadcrumb and sub-board rows show each board's H1
  title (§4), never the folder name alone.

**Keys:** `j/k` move, `enter` open (descend into a sub-board / open a task's detail),
`esc` back (up one board / close detail), `e` open the selected task in `$EDITOR`
(suspend TUI, resume on exit), `?` key-hint footer (`bubbles/help`), `q` quit.

**Mouse:** rows are BubbleZone hit-zones — click selects, double-click opens, wheel
scrolls. Scrolling is spring-smoothed with Harmonica; motion stays subtle, never slower
than the keyboard.

**Live refresh:** watch every directory in the tree with fsnotify. On any event, debounce
~100 ms, then **re-scan everything** — the tree is dozens of small files, so a full
re-read per refresh beats any incremental cache. Saving in `$EDITOR` or running a CLI
command in another terminal updates the view within a blink; no polling, no IPC.
(Folder-watching is the right idea: it's the standard mechanism, and re-scan-on-event
keeps the TUI stateless.) fsnotify watches are per-directory, not recursive: add one per
directory on the initial walk, and re-walk to pick up watches when a directory event
arrives (a new sub-board appears live).

**Read-only by design:** no status-cycling keybindings, no in-TUI forms. `$EDITOR` +
the CLI already cover every mutation, and the watcher makes them appear instantly.
Add TUI mutations only if that round-trip proves too slow in practice.

## 9. Implementation notes (Go)

- **One module, ports-and-adapters** — the layout and code rules live in `CLAUDE.md`:
  `cmd/duty` entrypoint; leaf `internal/names` (convention filenames, defined once);
  pure domain `internal/task`, `internal/board`; a filesystem port `internal/fsys`
  (OS + in-memory adapters); `internal/tree`/`internal/config` as queries over it;
  application services in `internal/app` (use-cases, sync invariant, never prints);
  presentation `internal/cli`, `internal/tui`; all tests in `tests/`.
- **Subcommands via `spf13/cobra`** — per-command help and usage for free; commands
  stay thin (parse → app service → format). Cobra's own printing is silenced so the
  contract holds: quiet on success, one lowercase stderr line + non-zero exit on error.
- **Deps:** `spf13/cobra` (CLI), `BurntSushi/toml` (config), `gopkg.in/yaml.v3` to *read*
  frontmatter robustly (lists!), `fsnotify/fsnotify` (watcher), and the Charm stack for the TUI — one
  ecosystem, not six separate deps to vet:
  - `bubbletea` — the TUI runtime (Elm-style update loop).
  - `bubbles` — stock components: `viewport` (detail scroll), `help` (key hints). Never
    hand-roll a component bubbles already ships.
  - `lipgloss` — all styling and layout; no raw ANSI anywhere.
  - `glamour` — detail-view markdown render.
  - `bubblezone` — mouse hit-zones for clickable rows.
  - `harmonica` — spring-smoothed scrolling.
  - `ntcharts` — the header status-distribution bar.
- **Writes are targeted line edits**, never YAML re-serialization — untouched bytes
  survive (re-serializing would reformat every file and break the round-trip invariant).
- **Writes are atomic:** temp file in the same directory, then rename over the target —
  the TUI's watcher (and any concurrent reader) never sees a half-written file.
- Frontmatter parse: `\A---\n(.*?)\n---\n` + `yaml.Unmarshal`. Status write:
  `(?m)^status: \S+`, first match only.
- Row find: the line containing `(<filename>)` that starts with `|`. Status cell update:
  `strings.Split(row, "|")`, replace `cells[len(cells)-2]`, rejoin — preserves spacing.
- Task-id → file: `filepath.WalkDir` from the tree root matching `<id>-*.md`; error names
  the id and notes archived tasks are read-only. Global NN uniqueness (§3) guarantees at
  most one match.
- Board discovery: `filepath.WalkDir` collecting directories that contain `BOARD.md`,
  skipping `archive/`. Reused by list, archive, the TUI scan, and NN numbering.
- Gate count: count `- [x]` vs `- [ ]` lines under `## Gates` (until the next `^## `).
- Companion agent-facing doc (`duty/README.md`, one page): the command table, the
  lifecycle→command mapping, and what stays the worker's judgment (filling
  Goal/Scope/Gates, ticking gates honestly, authoring report prose, respecting blocked-by).

## 10. Deliberately not built (YAGNI — add only when it hurts)

- **No state machine** on transitions — a flat setter + a written contract beats encoding
  workflow in code at this scale.
- **No due dates, priorities, assignees, labels, timestamps** — ordering on the board *is*
  the priority. Add a `created:` frontmatter line the day someone actually asks "how stale
  is this?".
- **No dependency enforcement** beyond existence-check at create — `blocked-by` is
  advisory; the worker contract enforces it.
- **No board regeneration command** — the board is edited surgically and drift is
  *surfaced* (`list`, TUI badge), which in practice keeps it honest without a rebuild path.
- **No editing archived tasks** — archive is read-only by convention; the CLI simply
  refuses to resolve ids there.
- **No TUI mutations** — viewer only; `$EDITOR` and the CLI are the write path.
- **No board delete/move commands** — tasks move (`duty move`); boards don't. Delete a
  board by deleting the folder and fixing the parent's `## Boards` bullet by hand; it's
  cosmetic anyway (tooling scans, never reads it).
- **No locking** — concurrent CLI writers can race on a `BOARD.md`. In practice: one
  human, agents run commands serially. Add an flock on the root when it actually hurts.
- **No stored rollups** — sub-board counts (`3/7 done`) are computed live from files by
  the TUI, written nowhere. Anything derived and stored is future drift.
- **No semantic config** — `duty.toml` tunes presentation (§7); statuses, naming, and
  board structure stay convention. A config key that changes file formats is a bug.
