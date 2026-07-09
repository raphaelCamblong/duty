---
id: T-05
title: TOML config loading
status: done
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
- [x] Precedence tests: default only, user only, project overriding user, partial
  files merging per-key; malformed TOML → error.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

### 2026-07-09 — done

Files changed:
- `internal/config/config.go` (new): `Config{Editor; TUI struct{Theme}}`,
  `Load(userPath, projectPath)`, `UserPath()`. Merge is sequential TOML decode
  into the defaults struct — a file only overrides the keys it sets, so per-key
  precedence (project ← user ← defaults) falls out of `toml.Unmarshal`.
  Missing files (and empty paths) are skipped via `fs.ErrNotExist`; any other
  read error or a TOML parse error is returned wrapped with `%w`.
- `tests/config_test.go` (new): table-driven `TestLoad` (11 subtests: defaults
  with/without `$EDITOR`, user only, project only, project-over-user, three
  partial-merge combinations, empty files, malformed user/project TOML → error)
  plus `TestLoadEmptyPaths` and `TestUserPath`. Fixtures via `t.TempDir()`,
  env via `t.Setenv`.

Gate output tails:
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok github.com/raphaelCamblong/duty/tests 0.499s coverage: 80.0% of statements in ./internal/...`
  (all `TestLoad` subtests PASS under `-v`).
- `gofmt -l .` → empty. `go vet ./...` → clean. `golangci-lint` not installed.

Deviations: none. Decisions within scope: an empty `$EDITOR` counts as unset
(falls back to `vi`); `UserPath` returns `(string, error)` because
`os.UserConfigDir` can fail; unknown TOML keys are ignored (only invalid TOML
is "malformed"). Root-only `duty.toml` enforcement left to T-04 as specified.
