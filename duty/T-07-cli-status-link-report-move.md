---
id: T-07
title: CLI status, link, report, move
status: done
blocked-by: [T-06]
---

# T-07 — CLI status, link, report, move

## Goal
The four mutation commands from spec §6, each maintaining the sync invariant in one
shot.

## Read first
`task-system-spec.md` §6 (`status`/`link`/`report`/`move` rows), §3–§5; `CLAUDE.md`.

## Scope
- `status <id> <status>` — frontmatter `status:` line + board row cell; reject unknown
  statuses; reject archived ids (read-only).
- `link <id> <section>` — move the row under `## <section>` (created above the footer
  if absent); prune any section left empty.
- `report <id>` — append stdin under `## Report` (heading created once); refuse empty
  stdin.
- `move <id> <board-path>` — path relative to tree root (`.` = root board); rename the
  file into the target board (same filename), drop source row + prune, append to the
  target's `Open tasks` (or `--section`), status preserved.

## Out of scope
archive/delete/list (T-08), TUI, any format change.

## Gates
- [x] Black-box tests per command: file+board both updated in one call; unknown
  status rejected; empty-stdin report rejected; move preserves status and filename
  and leaves both boards internally consistent; move to a non-existent board errors.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

Files changed: internal/cli/status.go, internal/cli/link.go, internal/cli/report.go,
internal/cli/move.go (new — finished a prior interrupted session's drafts after
critical review), internal/cli/cli.go, internal/cli/init.go, internal/cli/board.go,
internal/tree/tree.go (constant exports, see deviation below), tests/cli_mutate_test.go
(new, extended with one archived-id case for link).

Inherited work review: the four command files and cli_mutate_test.go left by the
interrupted session were already correct and complete against the Scope — status
rejects unknown statuses and archived ids without writing either file; link
creates/prunes sections and only ever touches the board (sections have no file-side
representation); report refuses empty/blank stdin and accumulates under one ## Report
heading; move renames the file into the target board, drops+prunes the source row,
appends the target row with status preserved, validates the board-path against the
tree root (rejects absolute paths, paths escaping the tree, and non-existent boards),
and handles the same-board case without a spurious rename. I kept all of it as-is
after reading every line against task-system-spec.md §3-6 and re-deriving each gate by
hand; the only gap I found was a missing "rejects archived ids" case for `link`
(status/report/move each had one, link did not) — added it.

Gate commands, tails:
  go build -o bin/duty ./cmd/duty            -> silent, exit 0
  go test ./tests/... -coverpkg=./internal/... -> ok  github.com/raphaelCamblong/duty/tests  0.3s  coverage: 86.9% of statements in ./internal/...
  gofmt -l .                                  -> empty
  go vet ./...                                -> clean
Manually smoke-tested the full loop in a scratch tree: init, create, board, status,
link, report, move all behaved as documented, including status rejecting an unknown
value and preserving in-progress through move.

Deviation (flagged by earlier review, applied here): centralized the convention
filenames into internal/tree, their one legitimate home — added exported
tree.BoardFile, tree.ConfigFile, tree.ArchiveDir, tree.ReadmeFile (previously
tree.go held boardFile/configFile as unexported duplicates of literals also hardcoded
in internal/cli's cli.go/init.go/board.go). cli.go's boardFile now aliases
tree.BoardFile instead of redeclaring the string; init.go and board.go's os.MkdirAll
calls now use tree.ArchiveDir, and init.go's README write uses tree.ReadmeFile. Left
one class of literal alone on purpose: internal/board's Render/AddBoardBullet embed
"BOARD.md"/"archive/"/"README.md" as literal markdown *content* (the rendered
convention line and footer/bullet hrefs), not filesystem operations — board is a pure
package forbidden from importing internal/tree (CLAUDE.md's dependency rule), so
these stay as content strings; noted here rather than silently diverging.

No new helpers were needed in task/board — CountGates/SetStatus/AppendReport and
SetRowStatus/MoveRow/DropRow/AddRow/PruneEmptySections already covered every case the
four commands needed.

Follow-ups left for later tasks: archive/delete/list are T-08; the master round-trip
invariant test (create -> status -> report -> link -> move -> move back -> delete ->
archive, byte-identical) needs those commands to exist and belongs with T-09/T-08.
