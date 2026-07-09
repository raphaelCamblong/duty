---
id: T-03
title: Board file domain package
status: todo
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
- [ ] Table-driven tests in `tests/` prove surgical edits: a board with hand-written
  prose and unusual spacing keeps every untouched line byte-identical through
  add/move/drop/status/prune/footer operations.
- [ ] Section create + prune covered; prune-never-default covered.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
