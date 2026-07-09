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

## Architecture

Layered, dependency rule points inward (clean architecture adapted to a Go CLI — the
same shape as `gh`/`hugo`: `cmd/` entrypoint, domain and infra under `internal/`).

```
cmd/duty/main.go     — entrypoint only: wire deps, dispatch, map error → exit code. ≤50 lines.
internal/
  task/              — domain, PURE (bytes in → bytes out): frontmatter parse, task
                       template render, targeted edits (status line, append report),
                       gate counting. Imports stdlib + yaml only.
  board/             — domain, PURE: BOARD.md model — row find/edit/add/drop, sections,
                       pruning, footer count, Boards bullets. Imports stdlib only.
  tree/              — infra: root & current-board resolution (walk-up), board discovery
                       (WalkDir, skip archive/), id → path, next NN.
  config/            — infra: TOML load, user < project precedence, root-only check.
  fsutil/            — infra: atomic write (temp + rename in same dir). Tiny.
  cli/               — presentation: subcommand dispatch (stdlib flag), handlers. The
                       sync invariant lives HERE: each mutating handler = task edit +
                       board edit, both through fsutil.
  tui/               — presentation: bubbletea program (model/update/view), fsnotify
                       watcher, tree-scan → view model.
tests/               — ALL test files. No _test.go anywhere else.
```

Dependency rule: `task` and `board` import no other internal package and touch no
filesystem — they transform bytes. `cli` and `tui` may import everything; nothing
imports `cli` or `tui`. If a change needs `task` to read a file, the change is in the
wrong layer.

## Code rules (mandatory)

- **Interfaces at the consumer, and only when a second implementation or a test double
  actually exists.** No speculative abstraction, no factory for one product.
- Errors: wrap with `fmt.Errorf("context: %w", err)`; sentinel `var Err…` only where a
  caller branches on it. Never `panic` outside `main`. User-facing errors are one
  lowercase line, no trailing period — `main` prints to stderr and exits 1 (spec: exit
  codes are the API).
- Quiet on success. Stdout is output, stderr is errors — never mix.
- Naming: no package stutter (`task.Task` yes, `task.TaskFile` no), short receivers,
  every exported symbol has a godoc comment starting with its name.
- Small functions, guard clauses, early return, no `else` after `return`.
- No package-level mutable state, no `init()`. Dependencies enter through constructors
  (`New…`) or parameters.
- **Line-surgical file edits** (spec invariant): find the target line, change only it,
  write back. Never parse-and-re-render a whole file. Never re-serialize YAML.
- Every write goes through `fsutil` (atomic). No raw `os.WriteFile` outside it.
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
