---
id: T-03
title: Board file domain package
status: done
blocked-by: [T-01]
---

# T-03 — Board file domain package

## Goal
`internal/board`: a pure model of `BOARD.md` from spec §4 — every operation is
line-surgical, never a re-render.

## Read first
`task-system-spec.md` §4 and the §6 behavioral invariants; `CLAUDE.md`.

## Scope
All functions take `content []byte`, return `[]byte` (+ error). No filesystem.
- `Render(title string) []byte` — skeleton: H1 = title, convention line, empty
  `## Open tasks` table, `Completed tasks (0) archived: …` footer.
- `Title(content) string` — the H1 text.
- `FindRow(content, filename)` — the `|`-prefixed line containing `(filename)`.
- `AddRow(content, section, id, filename, title, status)` — append to the section's
  table; create the section above the footer if absent.
- `SetRowStatus` — `strings.Split(row, "|")`, replace `cells[len(cells)-2]`, rejoin
  (preserves spacing).
- `MoveRow(content, filename, section)`, `DropRow(content, filename)`.
- `PruneEmptySections` — never removes `## Open tasks`.
- `SetArchivedCount(content, n)` — regex-rewrite the footer number.
- `AddBoardBullet(content, name, title)` — `## Boards` bullet, section created if absent.

## Out of scope
Filesystem access, knowledge of other boards or task-file internals.

## Gates
- [x] Table-driven tests in `tests/` prove surgical edits: a board with hand-written
  prose and unusual spacing keeps every untouched line byte-identical through
  add/move/drop/status/prune/footer operations.
- [x] Section create + prune covered; prune-never-default covered.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

2026-07-09 — done.

Files changed:
- `internal/board/board.go` (new): pure `BOARD.md` model. Exported: `Render`,
  `Title`, `FindRow`, `AddRow`, `SetRowStatus`, `MoveRow`, `DropRow`,
  `PruneEmptySections`, `SetArchivedCount`, `AddBoardBullet`, plus
  `DefaultSection` ("Open tasks"). Every edit splits on `\n`, touches only the
  target line(s), and rejoins — never a re-render. Imports stdlib only
  (`fmt`, `regexp`, `strconv`, `strings`); zero filesystem.
- `tests/board_test.go` (new): black-box table-driven tests. A fixture board
  with a hand-written banner, free prose, and a row with unusual cell spacing;
  every expected output is built from the same fixture lines, so any byte
  drift in an untouched line fails. Covers: skeleton render + title
  round-trip, row find (incl. filename-in-prose non-match), add to
  existing/new section, footerless-board error, status cell rewrite
  preserving odd spacing in other cells, drop, move (existing + created
  section, byte-identical row), prune (empty removed, default never, prose
  and bullets kept, multi-section pass), footer count rewrite, Boards bullet
  append + section creation, and `TestSurgicalRoundTrip` — add → status →
  move to a created section → move back → prune → drop → footer bump/restore
  ends byte-identical to the starting fixture.

Gate output tails:
- `go test ./tests/... -coverpkg=./internal/... -count=1` →
  `ok  github.com/raphaelCamblong/duty/tests  0.226s  coverage: 90.3% of statements in ./internal/...`
- `gofmt -l .` → empty; `go vet ./...` → clean. golangci-lint not installed.

Interface notes (for CLI consumers):
- `FindRow` returns `(row string, ok bool)` — comma-ok so `list` can branch
  on a missing row (drift) without a sentinel error.
- `MoveRow` does not prune; compose with `PruneEmptySections` (mirrors the
  scope listing them as separate operations).
- `AddRow`/`MoveRow` error when a new section is needed but the footer line
  is missing (the footer is the insertion anchor); `SetArchivedCount` errors
  on a missing footer too.
- `PruneEmptySections` removes only sections whose body is blank lines plus
  empty table scaffolding; sections holding prose or bullets are never
  touched, and `## Open tasks` is never removed.

Deviations: none. Follow-ups deliberately left: none.
