---
id: T-33
title: Justfile for developer experience
status: done
blocked-by: []
---

# T-33 — Justfile for developer experience

## Goal
`just <tab>` answers "how do I work on this repo": every canonical command has
a short recipe; CLAUDE.md stays the source of truth the recipes wrap.

## Read first
`CLAUDE.md` (Build & test — the canonical commands), `.goreleaser.yaml`,
`.github/workflows/ci.yml` (the recipes must match what CI runs).

## Scope
- `justfile` at the repo root:
  - `default` — `just --list`.
  - `build` — `go build -o bin/duty ./cmd/duty`.
  - `test` — the full suite with coverage; `race` — same under `-race`.
  - `lint` — golangci-lint via `go run` (the pinned repo config);
    `fmt` — `gofumpt -w .`; `vet`.
  - `tui` — build then run `bin/duty tui` on this repo's own tree.
  - `snapshot` — the GoReleaser dry run (`--snapshot --clean --skip=publish`).
  - `vuln` — govulncheck.
  - `check` — the pre-commit gate: fmt-check + vet + lint + test (what an
    agent runs before calling a task done).
  - `clean` — remove `bin/` and `dist/`.
- Recipes pass extra args positionally (`just test -run TestClaim` style,
  `"$@"` semantics) — never string interpolation.
- One-line pointers added: CLAUDE.md Build & test section ("`just` wraps
  these"), README dev corner (`just check`).
- Verify each recipe actually runs on this machine; `just --fmt --check` (or
  equivalent) clean if available.

## Out of scope
Replacing CI with just; make/task alternatives; install/release recipes beyond
`snapshot`; new tooling deps (just itself is assumed installed by the dev).

## Gates
- [x] Every recipe listed above runs successfully on this machine (tails in the
  report); `just` with no args lists them.
- [x] `just check` fails when a deliberate temp violation is introduced and
  passes clean (demonstrated in the report, then reverted).
- [x] Suite/lint unaffected (`go test ./tests/... -coverpkg=./internal/...
  -count=1` green; `golangci-lint run` 0 issues).
- [x] CLAUDE.md + README pointers added in the same change.

## Report

Added justfile at repo root with the 12 recipes from Scope (default, build,
test, race, lint, fmt, vet, tui, snapshot, vuln, check, clean). Positional
args pass through via `set positional-arguments := true` and literal "$@" in
each recipe body (verified `just test -run TestClaim -v` keeps a spaced
argument intact) — no string interpolation.

Verified every recipe on this machine (just 1.35.0 installed):
- `just` / `just default` list all 12 recipes via `just --list`.
- `build` produces bin/duty; `test` and `race` both green
  (coverage 86.6%, ~10s / ~12s).
- `vet` clean; `fmt` (gofumpt -w .) no-op on a clean tree; `lint`
  (golangci-lint v2.12.2 via go run) 0 issues.
- `tui` builds then launches ./bin/duty tui on this repo's own tree
  (confirmed the process starts and stays up under a real pty via `script`;
  the sandbox's plain exec has no /dev/tty, which is an exec-environment
  limit, not a recipe defect).
- `snapshot` runs the GoReleaser v2.17.0 dry run end to end (6 archives +
  homebrew cask written under dist/, skip=publish).
- `vuln` runs govulncheck: 0 vulnerabilities in code.
- `clean` removes bin/ and dist/.
- `just --fmt --check --unstable` reports no formatting diff on the
  justfile itself.

Gate demo: added a leading blank line inside internal/task/gate.go's Gates
function body (a gofumpt violation gofmt doesn't catch), ran `just check` —
failed with "gofumpt needs to be run on: internal/task/gate.go" and exit 1.
Reverted the edit (git diff confirms byte-identical), ran `just check`
again — vet clean, lint 0 issues, test green, exit 0.

CLAUDE.md Build & test section gained a one-line "`just` wraps these"
pointer; README gained a short "Hacking on it" section pointing at
`just check`.

Full suite and lint stayed green throughout:
go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, 86.6% coverage.
golangci-lint run -> 0 issues.
