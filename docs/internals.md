# Internals

## Implementation notes

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
  NN uniqueness (see tasks.md) guarantees at most one match.
- Board discovery: walk collecting directories containing `BOARD.md`, skipping
  `archive/`. Reused by reads, archive, the TUI scan, and NN numbering.
- Gate count: `- [x]` vs `- [ ]` lines under `## Gates` (until the next `^## `).
- Companion agent doc `duty/README.md` (one page): command table, lifecycle→command
  mapping, what stays the worker's judgment. Generated skeletons (task file, board,
  `duty/README.md`) are `go:embed`ed `.md.tmpl` templates (`text/template`).

## Deliberately not built (YAGNI — add only when it hurts)

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
- **No semantic config** — `duty.toml` tunes presentation (see config.md). A config
  key that changes file formats is a bug.
