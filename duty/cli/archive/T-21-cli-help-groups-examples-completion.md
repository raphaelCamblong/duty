---
id: T-21
title: CLI help groups, examples, completion
status: done
blocked-by: [T-20]
---

# T-21 — CLI help groups, examples, completion

## Goal
`duty --help` teaches the tool in one screen: grouped commands, an example on
every command, typo suggestions, shell completion, a version.

## Read first
`internal/cli` root command wiring; cobra features: `cobra.Group`, `Example`,
`SuggestionsMinimumDistance`, built-in `completion` command, `Version`.

## Scope
- Command groups on the root: `Author` (init, create), `Work` (status, report,
  move, archive, delete), `Read` (get), `Interface` (tui, completion).
- Root `Long`: the five-line lifecycle (get next → status in-progress → tick
  gates → status done + report → archive) so an agent reading `--help` learns
  the loop without the README.
- `Example:` on every command, real and copy-pasteable (`duty create task "Fix
  login" --blocked-by T-03`).
- Typo suggestions (`SuggestionsMinimumDistance: 2` — `duty creat` suggests
  `create`); cobra's `completion` command left enabled and listed under
  Interface.
- `--version` via cobra `Version` (default `dev`, overridable with
  `-ldflags "-X main.version=…"`).
- Error polish: resolve-failure errors gain a hint (`unknown task id "T-99" —
  try 'duty get tasks'`); still one lowercase line, still exit 1.
- Contract untouched: quiet success, exit codes, `--agent` formats.

## Out of scope
New commands or flags beyond `--version`; grammar changes; TUI.

## Gates
- [x] `duty --help` shows the four groups + lifecycle Long; every command's help
  shows an Example (spot-checked in tests for root, create task, get next).
- [x] `duty creat` exits 2 and suggests `create`; `duty --version` prints and
  exits 0; `duty completion zsh` emits a script.
- [x] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.

## Report

Files changed: cmd/duty/main.go, internal/cli/cli.go, internal/cli/{archive,create,delete,get,init,move,report,status,tui}.go, internal/tree/tree.go, task-system-spec.md, duty/BOARD.md, tests/cli_test.go, tests/cli_mutate_test.go, tests/cli_reads_test.go, tests/cli_help_test.go (new).

Picked up an interrupted agent's partial T-21 work, reviewed it critically against the task Scope, and found it essentially complete and correct: root command groups (Author/Work/Read/Interface), five-line lifecycle Long, Example on every command, SuggestionsMinimumDistance 2 on root and every dispatcher (with a custom unknownCommand helper wired to cobra's SuggestionsFor), completion left enabled and grouped under Interface, --version wired via a ldflags-overridable main.version var threaded through cli.Run, and a resolve-failure hint (unknown task id %q — try 'duty get tasks') centralized in tree.ResolveTask so every caller (status/report/move/archive/delete/get task) gets it uniformly. Verified by hand: duty --help renders all four groups + lifecycle; duty creat/get nex both exit 2 with correct "did you mean" suggestions; duty --version prints "dev" and honors -ldflags "-X main.version=..."; duty completion {bash,zsh,fish} all emit real scripts and exit 0; every leaf and group command's --help shows a copy-pasteable Examples: block; unknown-id errors are one lowercase line, exit 1, across all mutating commands.

Gate tails:
- gofmt -l . -> empty
- go vet ./... -> clean
- go build -o bin/duty ./cmd/duty -> ok
- go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, 83.6% coverage

Deviations: none from the task Scope. Spec updated in the same change (new "Help & discovery" paragraph plus the task-id resolution note) since behavior changed.
