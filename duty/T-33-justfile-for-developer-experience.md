---
id: T-33
title: Justfile for developer experience
status: todo
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
- [ ] Every recipe listed above runs successfully on this machine (tails in the
  report); `just` with no args lists them.
- [ ] `just check` fails when a deliberate temp violation is introduced and
  passes clean (demonstrated in the report, then reverted).
- [ ] Suite/lint unaffected (`go test ./tests/... -coverpkg=./internal/...
  -count=1` green; `golangci-lint run` 0 issues).
- [ ] CLAUDE.md + README pointers added in the same change.

## Report
