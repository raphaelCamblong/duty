---
id: T-16
title: Templated markdown rendering
status: todo
blocked-by: []
---

# T-16 — Templated markdown rendering

## Goal
Generated markdown (task skeleton, board skeleton, init README) is authored as
readable template files, not string-building code. Output stays byte-identical.

## Read first
`CLAUDE.md`, `internal/task` `Render`, `internal/board` `Render`, the README
generation in `internal/app`.

## Scope
- **Golden files first:** before touching any render, capture the current outputs as
  goldens in `tests/testdata/` (task skeleton with/without blocked-by and with a
  YAML-quoted title; board skeleton; init README) plus a test asserting renders
  match them. Commit-worthy proof the swap changes nothing.
- Swap each render to stdlib `text/template` with `go:embed`ed `.md.tmpl` files
  living in their owning package (`internal/task/task.md.tmpl`,
  `internal/board/board.md.tmpl`, `internal/app/readme.md.tmpl`). Template funcs
  for the conditional bits (YAML title quoting, blocked-by list). Templates parsed
  once at package init via `template.Must` (the one acceptable init-time work).
- Domain packages stay pure — `embed` is compile-time, no filesystem at runtime.
- Add one line to `task-system-spec.md` §9 noting skeletons are embedded templates.

## Out of scope
Any output change whatsoever (goldens are the contract); templating BOARD.md *edits*
(line surgery stays code); TUI.

## Gates
- [ ] Golden tests green: template output byte-identical to the pre-swap renders
  for every variant.
- [ ] Full suite green including `tests/roundtrip_test.go`
  (`go test ./tests/... -coverpkg=./internal/... -count=1`).
- [ ] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report
