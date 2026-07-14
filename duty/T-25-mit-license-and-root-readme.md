---
id: T-25
title: MIT license and root README
status: todo
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
- [ ] `LICENSE` present, MIT, correct holder/year.
- [ ] README renders clean (markdown lint by eye), every command in it verified
  copy-paste against `./bin/duty` in a scratch tree; no channel promised that
  T-26 doesn't deliver.
- [ ] Full suite still green; `gofmt -l .` empty (no code touched).

## Report
