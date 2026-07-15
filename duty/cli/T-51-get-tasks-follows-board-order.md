---
id: T-51
title: get tasks follows board order
status: todo
blocked-by: []
---

# T-51 — get tasks follows board order

## Goal
What you see is what agents get: `get tasks` lists rows in board order —
the same order `get next` walks — instead of filename order.

## Read first
`internal/app/list.go` (`boardRows` iterates `tree.TaskFileNames` = filename
order), `internal/app/get.go` (`nextInBoard` walks `board.Sections` order —
the reference behavior), the demo that exposed it: `move T-03 --top` changed
`get next`'s answer but not `get tasks`' display.

## Scope
- `List` iterates each board's rows in `board.Sections` order (files stay
  truth for status/title; the board supplies sequence) — mirroring
  `nextInBoard`. Tasks with a file but NO row (drift) append after the
  ordered rows, keeping their drift flag.
- Applies to human and `--agent` output identically (row order changes; TSV
  fields unchanged).
- SANCTIONED BEHAVIOR CHANGE: update only the tests that pin filename
  ordering; drift-flag tests unchanged.
- docs/cli.md `get tasks` line mentions board order = priority order.

## Out of scope
Sorting flags; TUI (already board-ordered); `get tracks` (path order is fine).

## Gates
- [ ] Scratch tree: `move T-03 --top` immediately changes `get tasks` output
  order; `get next` and the first `get tasks` row agree.
- [ ] A file-without-row still appears (flagged, after ordered rows).
- [ ] `just check` green; docs updated.
