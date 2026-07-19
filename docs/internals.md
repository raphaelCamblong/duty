# Internals

How the binary keeps the files honest. The layout and code rules live in
`CLAUDE.md`; this page is the behavior worth knowing.

## Implementation notes

- **One module, ports-and-adapters.** Cobra subcommands stay thin (parse → app
  service → format) with cobra's own printing silenced: quiet on success, one
  lowercase stderr line plus a non-zero exit on error.
- **Writes are targeted line edits, never YAML re-serialization — and atomic:** a
  temp file in the same directory, renamed over the target, so no read ever sees
  a half-written file.
- Frontmatter parse: `\A---\n(.*?)\n---\n` + `yaml.Unmarshal`. Status write:
  `(?m)^status: \S+`, first match only.
- Row find: the line containing `(<filename>)` that starts with `|`. Status
  cell: `strings.Split(row, "|")`, replace the second-to-last cell, rejoin —
  spacing preserved.
- Id → file: walk from the tree root matching `<id>-*.md`; global `NN`
  uniqueness guarantees at most one hit. An unknown id hints
  `try 'duty get tasks'`; an archived match notes archived tasks are read-only.
- Board discovery: collect directories holding a `BOARD.md`, skipping
  `archive/`. Reused by reads, archive, the TUI scan, and `NN` numbering.
- Gate count: `- [x]` vs `- [ ]` lines under `## Gates`, until the next `^## `.
- **Deps:** `spf13/cobra`, `BurntSushi/toml`, `gopkg.in/yaml.v3` (frontmatter
  read), `fsnotify`, `gofrs/flock`, and the Charm stack — `bubbletea`,
  `bubbles`, `lipgloss`, `glamour`, `bubblezone`, `harmonica`, `ntcharts`.
- Generated skeletons (task file, board, `duty/README.md`) are `go:embed`ed
  `.md.tmpl` templates rendered with `text/template`.

## Reading the tree

Every read runs through one loader. `app.Load` walks the tree once, reads each
board index and open task file a single time, and joins three truths into one
model — `TreeView` → `BoardView` → `SectionView` → `TaskView`: the file's
frontmatter, the board row that orders it, and the drift and waits computed from
what it just read. Every read surface projects over that one model — `get
tasks`, `get task`, `get next`, `get tracks`, the watcher's snapshots, and the
TUI — so the same tree can't answer one question two ways.

- **Drift is a typed class**, not a string — in sync, a status disagreement, no
  row, no file, or an unparsable file. Each surface renders the class its own
  way (a `⚠` badge with words in the TUI, a column in `get tasks`).
- **The dep oracle lives in memory.** A `blocked-by` id is met when its task is
  done or archived; anything else leaves the dependent waiting — a todo,
  in-progress, or blocked dep, or an id that resolves to no open file. A
  board-only row carries no file truth, so it never satisfies a dep. The CLI and
  the TUI read the same oracle, so a missing dependency blocks in both.
- Mutations never touch the model. Writes stay line-surgical (below); the loader
  is read-only, and the write path never assembles a `TreeView`.

## The write lock

Every mutating command (`create`, `status`, `report`, `set`, `gates`, `move`,
`archive`, `delete` — not `init`, not reads) holds one advisory `flock` on
`<root>/.duty.lock` for its whole duration. `get next --claim` re-scans and
marks the task `in-progress` under that same lock, so parallel claims each hand
back a distinct task. A writer that can't take the lock within 5s fails with
`tree is locked` rather than racing on a `BOARD.md`. Reads never lock.

:::caution
The lock is deliberately coarse — no per-task granularity, no lease or
heartbeat. A crashed agent leaves its task `in-progress`;
`duty status <id> in-progress --force` is the manual takeover.
:::

## Deliberately not built

YAGNI, with teeth — added only when a hardcoded value starts to hurt:

- **No state machine** on transitions — a flat setter plus a written contract.
- **No due dates, priorities, assignees, labels, timestamps** — board order *is*
  the priority.
- **No dependency enforcement** beyond an existence check at create —
  `blocked-by` is advisory; the worker contract enforces it.
- **No board regeneration** — boards are edited surgically and drift is
  surfaced, never rebuilt.
- **No editing archived tasks** — the CLI refuses to resolve ids there.
- **No TUI mutations** — viewer only. `$EDITOR` and the CLI are the write path;
  the watcher shows them instantly.
- **No board delete/move commands** — tasks move, boards don't. Delete a board
  by deleting the folder and fixing the parent's `## Boards` bullet by hand.
- **No stored rollups** — track counts are computed live from the files.
- **No semantic config** — `duty.toml` tunes presentation (see
  [Config](/config/)); a key that changes a file format is a bug.
