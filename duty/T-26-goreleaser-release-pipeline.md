---
id: T-26
title: GoReleaser release pipeline
status: todo
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
- [ ] Dry run passes locally:
  `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish`
  → `dist/` holds all six platform archives + checksums; archives contain
  binary, LICENSE, README, completions.
- [ ] A snapshot binary runs: `dist/` darwin_arm64 `duty --version` prints the
  snapshot version (ldflags wired).
- [ ] `go run …goreleaser… check` reports a valid config; both workflow YAMLs
  pass `actionlint` if available (else note it).
- [ ] Full suite still green; `gofmt -l .` empty.

## Report
