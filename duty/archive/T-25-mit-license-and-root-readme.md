---
id: T-25
title: MIT license and root README
status: done
blocked-by: []
---

# T-25 — MIT license and root README

## Goal
The repo has a landing page and a license: a root `README.md` that sells and
teaches duty in one screen-and-a-bit, and MIT terms every distro channel accepts.

## Read first
`task-system-spec.md` (source for the pitch), `duty/README.md` (the in-tree
convention doc — do NOT duplicate it, link it), `./bin/duty --help` output.

## Scope
- `LICENSE`: MIT, `Copyright (c) 2026 Raphael Camblong`.
- Root `README.md`, in this order:
  1. One-paragraph pitch: file-based task system — markdown task files + nested
     track boards, one binary, CLI + live TUI, built agent-first. No database,
     no daemon, greppable, diffable.
  2. A short real session (fenced): `duty init` → `duty create track api` →
     `duty create task "…" --blocked-by …` → `duty get next` → `duty status` →
     `duty archive`.
  3. **Install**: Homebrew tap (`brew install raphaelCamblong/tap/duty`),
     `go install github.com/raphaelCamblong/duty/cmd/duty@latest`, and
     prebuilt binaries from GitHub Releases. (apt/rpm deliberately later.)
  4. **The TUI**: `duty tui` — one tight paragraph on the master-detail view,
     live refresh, mouse/filtering.
  5. **For agents**: `get next` / `get tasks --agent` TSV, quiet+exit-code
     contract, link to `duty/README.md` (convention) and `task-system-spec.md`.
  6. Dev corner: build/test one-liners, link to `CLAUDE.md`.
- Honest about pre-release state: install lines that don't work until the first
  tag are marked "(after v0.1.0)".
- No badges, no screenshots yet (a TUI GIF is a nice later; note it as a
  follow-up in the report).

## Out of scope
GoReleaser/CI (T-26); apt/rpm; docs hosting (T-23); touching `future.md`.

## Gates
- [x] `LICENSE` present, MIT, correct holder/year.
- [x] README renders clean (markdown lint by eye), every command in it verified
  copy-paste against `./bin/duty` in a scratch tree; no channel promised that
  T-26 doesn't deliver.
- [x] Full suite still green; `gofmt -l .` empty (no code touched).

## Report

Done. Added the repo's landing page and license.

Files added:
- /Users/raphael/perso/duty/LICENSE — MIT, Copyright (c) 2026 Raphael Camblong.
- /Users/raphael/perso/duty/README.md — root README, six sections in the required
  order: pitch, a verified real session, install, the TUI, for agents, development.

Verification (scratch tree via mktemp -d, outside the repo):
- Every command shown was run copy-paste against ./bin/duty: init, cd duty,
  create track api, create task, create task --blocked-by T-01, get next,
  status in-progress, status done, archive, plus get tasks/get next --agent.
  The get next output block in the README is byte-accurate to real output (only
  the temp path prefix trimmed to ~/payments for readability).
- go build -o bin/duty ./cmd/duty verified (the from-source install line).
- Install channels that need the first tag (Homebrew tap, go install @latest,
  GitHub Releases prebuilt binaries) are marked "(after v0.1.0)"; apt/rpm noted
  as deliberately later. No channel is promised that T-26 doesn't deliver.

Gates:
- gofmt -l . empty; go vet ./... clean (no code touched).
- go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, 85.3% coverage.

The convention doc (duty/README.md) and the spec (task-system-spec.md) are linked,
not duplicated. No badges, no screenshots.

Follow-up deliberately left: a TUI demo GIF/asciinema for the TUI section (a nice
later once GoReleaser/CI lands in T-26 and there's a tagged build to record).
