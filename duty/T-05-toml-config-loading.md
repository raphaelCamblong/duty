---
id: T-05
title: TOML config loading
status: todo
blocked-by: [T-01]
---

# T-05 — TOML config loading

## Goal
`internal/config` per spec §7: defaults ← user config ← project config, zero-config
always works.

## Read first
`task-system-spec.md` §7; `CLAUDE.md`.

## Scope
- `Config{ Editor string; TUI struct{ Theme string } }`.
- `Load(userPath, projectPath string) (Config, error)` — merge project over user over
  defaults; a missing file is not an error; a malformed file is.
- Defaults: `Editor` = `$EDITOR`, else `vi`; `Theme` = `auto`.
- `UserPath()` — `os.UserConfigDir()/duty/config.toml`.
- Dependency: `github.com/BurntSushi/toml`.

## Out of scope
Writing config, any key beyond `editor` and `tui.theme`, root-only enforcement
(that's `tree`'s job, T-04).

## Gates
- [ ] Precedence tests: default only, user only, project overriding user, partial
  files merging per-key; malformed TOML → error.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
