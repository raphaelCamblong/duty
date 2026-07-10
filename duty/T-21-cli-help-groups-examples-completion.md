---
id: T-21
title: CLI help groups, examples, completion
status: todo
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
- [ ] `duty --help` shows the four groups + lifecycle Long; every command's help
  shows an Example (spot-checked in tests for root, create task, get next).
- [ ] `duty creat` exits 2 and suggests `create`; `duty --version` prints and
  exits 0; `duty completion zsh` emits a script.
- [ ] Full suite green; `gofmt -l .` empty; `go vet ./...` clean; build ok.

## Report
