# duty

File-based task system: markdown task files + nested board indexes + one Go binary
(CLI + TUI). **`task-system-spec.md` is the source of truth for behavior** — read it
before writing any code. This file governs how the code is written. On conflict: the
spec wins on behavior, this file wins on style and structure.

## Build & test

- Go, latest stable. Module: `github.com/raphaelCamblong/duty`.
- Build: `go build -o bin/duty ./cmd/duty` (`bin/` is gitignored; a bare
  `go build ./cmd/duty` would try to write an executable named like the `duty/` folder)
- Test: `go test ./tests/... -coverpkg=./internal/...`
- Before considering any change done: `gofmt -l .` empty, `go vet ./...` clean, all
  tests green. Run `golangci-lint run` if installed.
- `just` wraps these — `just <tab>` lists every recipe, `just check` is the
  pre-commit gate.

## Architecture

Ports-and-adapters around a pure domain (clean architecture; dependencies point
inward). `gh`/`hugo`-shaped on the outside: `cmd/` entrypoint, everything under
`internal/`.

```
cmd/duty/main.go     — entrypoint only: wire fsys.OS into the app, delegate to cli. ≤50 lines.
internal/
  names/             — leaf vocabulary: the convention's file/dir names (BOARD.md,
                       duty.toml, README.md, archive, duty). Zero deps; imported by all.
                       A filename literal outside this package is a bug.
  task/              — domain, PURE (bytes in → bytes out): frontmatter, template,
                       targeted edits, gate counting, slugs. stdlib + yaml + names only.
  board/             — domain, PURE: BOARD.md model, line-surgical. stdlib + names only.
  fsys/              — the filesystem PORT: FS interface (read, atomic write, rename,
                       remove, mkdir, readdir, walk) + adapters: OS (real, atomic
                       temp+rename writes) and Mem (in-memory test double).
  tree/              — repository: discovery, walk-up, id resolution, numbering —
                       queries over an injected fsys.FS.
  config/            — TOML load over fsys.FS; user < project precedence.
  app/               — application services: one use-case per verb (Init, CreateTask,
                       CreateBoard, SetStatus, Link, Report, Move, Archive, Delete,
                       List). App{FS} constructor-injected; returns data — never
                       prints, never parses flags. The sync invariant lives HERE.
  cli/               — presentation: cobra commands, thin — parse flags, call app,
                       format output (human or --agent TSV), map errors → exit codes.
  tui/               — presentation: bubbletea program over the same scan/app layers.
tests/               — ALL test files. No _test.go anywhere else.
```

Dependency rule (inward only): `names` ← domain ← {`fsys`, `tree`, `config`} ← `app` ←
{`cli`, `tui`}. Nothing imports `cli` or `tui` (main delegates; `cli` launches `tui`).
If code outside `internal/fsys` calls `os.*` file APIs, the change is in the wrong
layer — it goes through the port.

## Code rules (mandatory)

- **Interfaces at the consumer, and only when a second implementation or a test double
  actually exists.** No speculative abstraction, no factory for one product.
- Errors: wrap with `fmt.Errorf("context: %w", err)`; sentinel `var Err…` only where a
  caller branches on it. Never `panic` outside `main`. User-facing errors are one
  lowercase line, no trailing period — `main` prints to stderr and exits 1 (spec: exit
  codes are the API).
- Quiet on success. Stdout is output, stderr is errors — never mix.
- Naming: no package stutter (`task.Task` yes, `task.TaskFile` no), short receivers,
  every exported symbol has a godoc comment starting with its name — exactly one terse
  line. NO narrating inline comments: if a block needs explanation, extract a named
  function; comment only a constraint the code cannot express.
- Small functions, guard clauses, early return, no `else` after `return`.
- No package-level mutable state, no `init()`. Dependencies enter through constructors
  (`New…`) or parameters.
- **Line-surgical file edits** (spec invariant): find the target line, change only it,
  write back. Never parse-and-re-render a whole file. Never re-serialize YAML.
- Every filesystem touch goes through `fsys.FS` (writes atomic). No `os.*` file calls
  outside `internal/fsys`.
- CLI commands are cobra and stay thin: parse → app service → format. Business logic
  in a cobra `RunE` is a wrong-layer bug.
- Concurrency exists only in the TUI watcher. Everything else is synchronous.
- TUI: all styling through lipgloss (no raw ANSI); use bubbles components before
  hand-rolling; keep `update` logic pure so it's testable without a terminal.

## Testing (mandatory)

- All tests live in `tests/`, one package (`package tests`), black-box: import the
  internal packages and exercise their exported API. Shared helpers go in
  `tests/helpers_test.go`; golden files in `tests/testdata/`.
- Table-driven with `t.Run` subtests; filesystem fixtures via `t.TempDir()` — never
  write into the repo.
- CLI behavior is tested by calling the `cli` package entry with args + a temp tree,
  not by exec'ing a built binary.
- **The master test is the spec's round-trip invariant** (§6): create → status →
  report → link → move → move back → delete → archive leaves the tree byte-identical
  (hash before/after). Every behavioral invariant in spec §6 gets a test.
- TUI: test the pure parts (scan → view model, update transitions). Don't test rendering.

## Process

- The spec is the full scope. Build each piece **complete** — no v1/v2/v3 passes, no
  `TODO` stubs, no placeholder implementations. A task is done when its gates pass.
- Deviations from the spec are spec bugs: fix `task-system-spec.md` in the same change,
  don't silently diverge.
