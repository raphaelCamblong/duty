---
id: T-09
title: Spec invariants test suite
status: done
blocked-by: [T-08]
---

# T-09 — Spec invariants test suite

## Goal
Encode spec §6's behavioral invariants as the master acceptance tests, and fix any
violation they surface in the same task.

## Read first
`task-system-spec.md` §6 "Behavioral invariants"; `CLAUDE.md` testing rules.

## Scope
- `tests/roundtrip_test.go` — THE master test: on a scratch tree, run
  create → status → report → link → move → move back → delete → archive and assert
  the `duty/` tree hashes byte-identical before/after.
- Surgical-edit proof at the CLI level: a board salted with hand-written prose,
  banners, and odd spacing survives every command byte-identically outside the
  target line.
- Prune never removes `## Open tasks`; `list` never writes.
- Bugs found are fixed here — in `internal/`, or in `task-system-spec.md` if the spec
  itself is wrong (same change).

## Out of scope
New features, TUI, refactors beyond what a fix requires.

## Gates
- [x] Round-trip test green (tree hash identical).
- [x] Salted-board test green through all mutating commands.
- [x] Full suite `go test ./tests/... -coverpkg=./internal/...` green;
  `gofmt -l .` empty; `go vet ./...` clean.

## Report

Files changed: `tests/roundtrip_test.go` (new; only file touched — no violations
surfaced in `internal/` or the spec, so nothing else needed fixing).

What it encodes (spec §6, driven end-to-end through `cli.Run` on `t.TempDir()` trees):
- `TestRoundTrip` — the master test. A hand-salted tree (prose, banner, HTML comment,
  odd cell padding, sub-board, two archived tasks) is sha256-hashed over every
  directory and file, then a scratch task runs create → status → report → link (new
  section) → move → move back → delete → archive; the hash is byte-identical after.
  A mid-way hash check guards against a hash function that can't see changes.
- `TestSaltedBoardSurvivesEveryMutation` — one subtest per mutating command
  (status, create, report, link ×2, move ×2, delete, archive, board). Each asserts
  the ENTIRE tree file-by-file against an expectation built from the fixture
  constants via single-occurrence string replacement, so a command that touches one
  byte outside its target line fails with a named file and a diff.
- `TestPruneNeverRemovesDefaultSection` — delete- and archive-driven pruning on a
  board whose `## Open tasks` is already empty: `## Later` goes, the default stays.
- `TestListNeverWrites` — list/`--agent`/`--status` over a board salted with both
  drift kinds (row says done, row missing): drift flagged on stdout, tree hash
  unchanged.

Suite teeth verified by mutation: three violations injected by hand (prune removes
the default section; SetRowStatus normalizes the whole row; list auto-heals drift)
each made the relevant test fail with an exact byte diff, then were reverted.

Gate tails:
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok github.com/raphaelCamblong/duty/tests  coverage: 86.1% of statements in ./internal/...`
- `gofmt -l .` → empty; `go vet ./...` → clean.

Deviations: none. One documented nuance encoded as a test rather than a fix: a
cross-board `move` re-renders the moved row from the task file (spec §6 — the file
is truth), so hand padding INSIDE that one row does not survive move-there-and-back;
every byte outside it must, and the test pins exactly that.
