---
id: T-06
title: CLI dispatch, init, create, board
status: done
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
- [x] Black-box tests drive `cli.Run` in `t.TempDir()`: init bootstraps then refuses
  re-init inside the tree; create writes file AND row in one call, rejects unknown
  `--blocked-by`; board creates the skeleton and the parent bullet.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

### 2026-07-09 — done

Files changed:
- `internal/cli/cli.go` (new): `Run(args, stdin, stdout, stderr) int` — switch
  dispatch, `flag.NewFlagSet` per command (no cobra); missing/unknown command →
  one-line stderr, exit 2; handler errors → one lowercase stderr line, exit 1.
  Shared helpers: `positionals` (loops `fs.Parse` so flags work before AND after
  the title, matching the spec's `duty create <title> [--slug S]…` usage;
  `flag.ErrHelp` maps to the command's usage line), `stringList` (repeatable +
  comma-separated `--blocked-by`), `nameRE` (`^[a-z0-9-]+$`).
- `internal/cli/init.go` (new): `duty init [title]` — refuses when
  `tree.CurrentBoard(cwd)` resolves (inside a tree, or `./duty` already exists);
  creates `duty/` + `archive/`, writes `board.Render(title)` (default `Board`)
  and the §9 convention README (command table, lifecycle→command mapping,
  worker's-judgment section) via `fsutil.WriteAtomic`.
- `internal/cli/create.go` (new): `duty create <title> [--slug S]
  [--blocked-by ID]… [--section NAME]` — current board via `tree.CurrentBoard`,
  root via `tree.FindRoot`; every `--blocked-by` id validated tree-wide with
  `tree.ResolveTask` (`ErrArchived` counts as existing — an archived dep is a
  satisfied one); `tree.NextNN` numbers tree-wide; sync invariant in one shot:
  the new board bytes (`board.AddRow`, status `todo`) are computed BEFORE any
  write, then task file (`task.Render`) and board are written atomically; the
  created path is the only output.
- `internal/cli/board.go` (new): `duty board <name> [--title T]` — name
  validated against `[a-z0-9-]+`, refuses an existing folder; parent bullet
  bytes (`board.AddBoardBullet`) computed before creating `<name>/` +
  `archive/` + skeleton `BOARD.md`; all writes via `fsutil.WriteAtomic`.
- `cmd/duty/main.go`: thin delegate —
  `os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`.
- `tests/cli_test.go` (new): black-box, drives `cli.Run` (never the binary)
  from `t.TempDir()` via `t.Chdir`; 30 subtests: dispatch (no/unknown command),
  init (skeleton bytes == `board.Render`, custom H1, refuses re-init from
  beside and inside the tree, extra-arg usage error), create (file AND row
  after one call, path is the only stdout line, flags after title, NN skips an
  archived T-07, blocked-by resolves across boards, unknown blocked-by writes
  nothing, archived blocked-by accepted, `--section` inserts above the footer,
  validation table), board (skeleton + parent bullet, default title, nesting,
  existing-folder refusal with parent untouched, name-validation table).

Gate output tails:
- `go test ./tests/... -coverpkg=./internal/... -count=1` →
  `ok github.com/raphaelCamblong/duty/tests 0.509s coverage: 88.6% of
  statements in ./internal/...` (all subtests PASS under `-v`).
- `gofmt -l .` → empty; `go vet ./...` → clean;
  `go build -o bin/duty ./cmd/duty` → exit 0. golangci-lint not installed.

Decisions within scope (no spec deviations):
- Exit codes: dispatch-level usage problems (no command, unknown command)
  exit 2, handler errors exit 1 — spec only requires `≠0` + one-line stderr.
- `create` prints the created path absolute (derived from the resolved board
  dir) — unambiguous for agents regardless of cwd.
- Handler failure ordering favors the invariant: all reads/validation and the
  new board bytes happen before the first write, so a refused create/board
  changes nothing (tests assert byte-identical boards after refusals).
- `--slug` is validated against the same `[a-z0-9-]+` class as board names
  (a slug with `/` or spaces would break filenames and row lookup); an
  unslugifiable title without `--slug` is an error naming the flag.
- `--blocked-by` accepts repeats and comma-separated lists; an archived id
  passes the existence check via `errors.Is(err, tree.ErrArchived)`.

Follow-ups deliberately left: none. status/link/report/move are T-07,
archive/delete/list/--agent are T-08 (`Run`'s switch grows a case each).
