---
id: T-51
title: get tasks follows board order
status: done
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
- [x] Scratch tree: `move T-03 --top` immediately changes `get tasks` output
  order; `get next` and the first `get tasks` row agree.
- [x] A file-without-row still appears (flagged, after ordered rows).
- [x] `just check` green; docs updated.

## Report

### 2026-07-16 12:21 — done

List now iterates each board's rows in board.Sections order (mirroring
nextInBoard) instead of filename order; files stay truth for status/title.
A task file with no board row (drift) is appended after the ordered rows,
still flagged. Applies identically to human and --agent output. Verified in
a scratch tree: `move T-03 --top` immediately reorders `get tasks`' output
to match `get next`'s answer, and a row-less file still appears, flagged,
at the end. docs/cli.md's `get tasks` line now notes board order = priority
order. No existing test pinned filename order, so none needed updating;
`just check` is green.
