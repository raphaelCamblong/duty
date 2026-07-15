---
id: T-48
title: Reorder board rows from the CLI
status: done
blocked-by: []
---

# T-48 — Reorder board rows from the CLI

## Goal
Board order is priority — so the CLI can finally set it: `duty move <id>
--top | --before <ref> | --after <ref>` relocates a row line-surgically.
The last reason to hand-edit a BOARD.md disappears.

## Read first
`internal/board` (row ops are line-surgical; MoveRow exists for sections),
`internal/app/move.go`, docs/cli.md move section, the round-trip invariants
in tests/roundtrip_test.go.

## Scope
- `internal/board` gains a pure reorder op: relocate an existing row to the
  top of a section, or to immediately before/after another row, preserving
  the moved row's bytes exactly (it is the same line, moved).
- `duty move <id> --top` — row to the top of its section. `--before <ref>` /
  `--after <ref>` — row adjacent to <ref>'s row; if <ref> sits in another
  section of the same board, the row adopts that section. <ref> in a
  different board → error (`move --track` first). Position flags are
  mutually exclusive with each other, combinable with --track/--section
  (position applies after the relocation).
- App-level under the write lock, same sync guarantees; status/file
  untouched (board-only edit — the one command that legitimately edits only
  the board, like link did; note it in docs/internals.md's invariants line
  if needed).
- Tests: byte-identity outside the moved line; top/before/after each; the
  cross-section adopt; the cross-board error; flag exclusivity; round-trip
  extended with a reorder + reorder back.
- `docs/cli.md` move section updated.

## Out of scope
Reordering tracks in the Boards list; multi-row moves; TUI reordering.

## Gates
- [x] Scratch tree: --top/--before/--after each verified, byte-identical
  outside the moved line (test-proven); reorder + inverse reorder restores
  the exact original board bytes.
- [x] `just check` green; docs updated; existing tests unedited except the
  round-trip extension.

## Report

Added a pure line-surgical reorder op to internal/board (ReorderTop/Before/After: the moved row keeps its exact bytes) and duty move --top | --before REF | --after REF at the app layer under the write lock, board-only edit. Position flags are mutually exclusive, combine with --track/--section, and --before/--after adopt the ref row section within the same board; a ref in another board errors. Tests: board byte-identity + reorder round-trip, salted-board top/before/after + cross-section adopt, CLI cross-board error and flag exclusivity, and the master round-trip extended with a reorder + inverse. docs/cli.md move section updated. just check green.
