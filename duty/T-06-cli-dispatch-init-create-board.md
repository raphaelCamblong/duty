---
id: T-06
title: CLI dispatch, init, create, board
status: todo
blocked-by: [T-02, T-03, T-04]
---

# T-06 — CLI dispatch, init, create, board

## Goal
`internal/cli`: subcommand dispatch wired into `main`, plus the three creation
commands from spec §6. The sync invariant (task file + board row in one command)
lives in this package.

## Read first
`task-system-spec.md` §6 (`init`/`create`/`board` rows + agent-output paragraph),
§2–§4; `CLAUDE.md` (cli layer, error style).

## Scope
- `cli.Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` —
  `flag.NewFlagSet` per command; unknown command → one-line stderr, exit ≠ 0.
  `cmd/duty/main.go` becomes a thin delegate to it.
- `init [title]` — create `duty/` in cwd: skeleton `BOARD.md` (H1 = title, default
  `Board`), `README.md`, `archive/`. Refuse inside an existing tree.
- `create <title> [--slug S] [--blocked-by ID…] [--section NAME]` — current board;
  validate blocked-by ids tree-wide; NN tree-wide; write template + board row (`todo`);
  print the created path (only output).
- `board <name> [--title T]` — validate `[a-z0-9-]+`; skeleton `BOARD.md`
  (H1 = title, default the name) + `archive/`; bullet in parent's `## Boards`;
  refuse existing folder.
- Every write through `fsutil.WriteAtomic`. Quiet on success (except create's path).

## Out of scope
status/link/report/move (T-07), archive/delete/list (T-08), TUI, config wiring.

## Gates
- [ ] Black-box tests drive `cli.Run` in `t.TempDir()`: init bootstraps then refuses
  re-init inside the tree; create writes file AND row in one call, rejects unknown
  `--blocked-by`; board creates the skeleton and the parent bullet.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
