# Tracks & boards

A track is a folder with its own board. They nest, all the way down, and each
one is self-contained: its own tasks, its own `archive/`.

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

The **current board** is the nearest ancestor of your cwd that holds a
`BOARD.md` — a git-style walk-up (from outside the tree, it falls back to
`./duty/`). Creating commands target the current board; id-taking commands
resolve tree-wide.

## The board file

`BOARD.md` is the one place **ordering** lives — files can't express priority —
and the one page showing a board's state at a glance.

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

- The **H1 is the board's title** (root default `Board`; tracks get theirs from
  `create track`) — the truth for the TUI breadcrumb and track rows. The Task
  cell links a bare id → filename; row lookup keys on the `(filename)` substring.
- **Three columns, no creep.** Status is the one denormalization, kept honest by
  the sync invariant; everything else is read from the files.
- **`## Boards`** bullets (`- [name/](name/BOARD.md) — Title`) are for humans.
  `create track` appends one; tooling never reads them, and they're omitted when
  there are no tracks.
- **Sections** are `## <Name>` headers, each with its own table. `## Open tasks`
  always exists; others are created on demand and pruned when empty. New
  sections insert *above* the footer, whose count `archive` rewrites.

:::note
Order top-to-bottom **is** the priority — `get next` and the TUI both read the
board in build order. There are no due dates or priority fields; the row order
is the signal.
:::
