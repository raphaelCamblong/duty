---
id: T-29
title: Track path addressing across commands
status: todo
blocked-by: []
---

# T-29 — Track path addressing across commands

## Goal
Any board-scoped command can address a track by path instead of by cwd:
`duty create task "x" --in api/auth` works from anywhere in the tree.

## Read first
`task-system-spec.md` §2 (current-board resolution) and §6; `internal/app`
(`walkBoards`, `CurrentBoard` usage); how `move --track` already validates a
root-relative path (`app/move.go`).

## Scope
- **One flag, one mechanism:** `--in <track-path>` (long-only) — root-relative
  slash path, `.` = root board. Semantics: resolve the tree root from cwd as
  today, then current board := `<root>/<path>`. Must be an existing board;
  error `unknown track "api/auth"` otherwise (reuse/share `move --track`'s
  path validation — one validator, one error shape).
- Added as a LOCAL flag (shared `addInFlag(cmd)` helper, one honest help
  string) on exactly the commands with board context: `create task`,
  `create track` (the new track is created under the `--in` board),
  `get tasks`, `get tracks`, `get next`, `archive`. Id-addressed commands
  (`status`, `report`, `move`, `delete task`) don't get it — ids already
  resolve tree-wide. `move` keeps its own `--track` (destination) untouched.
- App layer: one shared resolution helper (e.g. `contextBoard(cwd, in)
  (boardDir string, err error)`) that every affected method threads through —
  no per-command reimplementation.
- Docs in the same change: spec §6 (flag on the affected rows + one paragraph
  under the table), root README session gains one `--in` line, generated
  readme template + golden updated.
- Tests: black-box per command — `--in` from outside any board dir, `--in .`,
  nested `--in api/auth` (3 levels), unknown path error, and `create track x
  --in api` creating `api/x/` with the bullet in `api/BOARD.md`.

## Out of scope
Claiming/locking (T-30); path support on id-addressed commands; `-C`-style
global chdir; TUI.

## Gates
- [ ] Scratch-tree: every listed command works with `--in` from the tree root
  AND from outside the tree (cwd-independent), 3-level path included.
- [ ] Unknown-path error is one lowercase line naming the path, exit 1.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` clean; `gofumpt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok.
- [ ] Spec §6, README, template + golden updated together.

## Report
