# Tracks & boards

## Layout

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

## Boards

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
