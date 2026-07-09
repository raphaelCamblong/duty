---
id: T-09
title: Spec invariants test suite
status: todo
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
- [ ] Round-trip test green (tree hash identical).
- [ ] Salted-board test green through all mutating commands.
- [ ] Full suite `go test ./tests/... -coverpkg=./internal/...` green;
  `gofmt -l .` empty; `go vet ./...` clean.

## Report
