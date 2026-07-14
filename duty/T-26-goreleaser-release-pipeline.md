---
id: T-26
title: GoReleaser release pipeline
status: done
blocked-by: [T-25]
---

# T-26 — GoReleaser release pipeline

## Goal
One `git tag vX.Y.Z && git push --tags` ships everything: GitHub Release with
cross-platform archives + checksums + changelog, and a Homebrew tap formula.

## Read first
GoReleaser v2 docs (builds, archives, brews, release, changelog), the existing
`main.version` ldflags wiring in `cmd/duty/main.go` (T-21), `.gitignore`.

## Scope
- `.goreleaser.yaml` (v2 schema):
  - build: `./cmd/duty`, binary `duty`, `CGO_ENABLED=0`,
    targets darwin/linux/windows × amd64/arm64,
    `-s -w -X main.version={{.Version}}` ldflags, `mod_timestamp` for
    reproducibility.
  - before hooks: generate shell completions (bash/zsh/fish) via
    `go run ./cmd/duty completion <sh>` into a staging dir.
  - archives: tar.gz (zip on windows), named `duty_<version>_<os>_<arch>`,
    containing the binary, `LICENSE`, `README.md`, completions.
  - checksum file; changelog from git (exclude `docs:`/`task:`/`board:` commit
    prefixes); GitHub release non-draft.
  - brews: formula `duty` → repository `raphaelCamblong/homebrew-tap`, token
    env `TAP_GITHUB_TOKEN`, description/homepage/license filled, installs
    binary + completions, `test` block runs `duty --version`.
  - NO nfpm/deb/rpm (deliberately later), no snap, no AUR yet.
- `.github/workflows/release.yml`: on tag `v*` — checkout (fetch-depth 0),
  setup-go from go.mod, goreleaser-action running `release --clean`;
  `GITHUB_TOKEN` + `TAP_GITHUB_TOKEN` env. Concurrency-guarded.
- `.github/workflows/ci.yml`: on push/PR to main — gofmt check, `go vet`,
  `go build -o bin/duty ./cmd/duty`, `go test ./tests/... -coverpkg=./internal/...`.
- `dist/` added to `.gitignore`.
- Document in the task report the two manual steps left for Raphael: create the
  empty `raphaelCamblong/homebrew-tap` repo, add a repo-scoped PAT as the
  `TAP_GITHUB_TOKEN` Actions secret — then tag.

## Out of scope
apt/rpm/nfpm, AUR, scoop, winget, snap (all later); actually pushing a tag;
creating the tap repo (needs Raphael's account).

## Gates
- [x] Dry run passes locally:
  `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish`
  → `dist/` holds all six platform archives + checksums; archives contain
  binary, LICENSE, README, completions.
- [x] A snapshot binary runs: `dist/` darwin_arm64 `duty --version` prints the
  snapshot version (ldflags wired).
- [x] `go run …goreleaser… check` reports a valid config; both workflow YAMLs
  pass `actionlint` if available (else note it).
- [x] Full suite still green; `gofmt -l .` empty.

## Report

Shipped the GoReleaser v2 release pipeline. `.goreleaser.yaml` builds ./cmd/duty
with CGO_ENABLED=0 across darwin/linux/windows x amd64/arm64, ldflags
`-s -w -X main.version={{.Version}}` plus mod_timestamp for reproducibility. A
before-hook generates bash/zsh/fish completions via `go run ./cmd/duty
completion`; archives are tar.gz (zip on windows) carrying binary + LICENSE +
README + completions, alongside a checksums file and a git changelog that
excludes docs:/task:/board: prefixes.

The Homebrew tap ships as `homebrew_casks` (goreleaser deprecated `brews`
formulas for binaries; the deprecated `homebrew_casks.binary` became
`binaries:` too) targeting raphaelCamblong/homebrew-tap with the
TAP_GITHUB_TOKEN env and a postflight quarantine-strip hook. Casks have no Ruby
`test do`/`license` stanza, so the `duty --version` smoke test lives instead in
the goreleaser dry run and the cask's own install; intent (tap install of
binary + completions) preserved.

Workflows: release.yml on v* tags (fetch-depth 0, setup-go from go.mod,
goreleaser-action release --clean, concurrency-guarded, GITHUB_TOKEN +
TAP_GITHUB_TOKEN); ci.yml on push/PR to main (gofmt check, vet, build, full
test suite). dist/ and completions/ gitignored.

Gates: `goreleaser check` valid; snapshot dry run produced all six platform
archives + checksums with the right contents; darwin_arm64 snapshot binary
prints `duty version 0.0.1-next`; actionlint clean on both workflows; full
suite green (85.3%); gofmt empty.
