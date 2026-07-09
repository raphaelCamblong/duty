---
id: T-16
title: Templated markdown rendering
status: done
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
- [x] Golden tests green: template output byte-identical to the pre-swap renders
  for every variant.
- [x] Full suite green including `tests/roundtrip_test.go`
  (`go test ./tests/... -coverpkg=./internal/... -count=1`).
- [x] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

Templated the three generated skeletons via text/template + go:embed, output byte-identical.

Files changed:
- internal/task/task.go, internal/task/task.md.tmpl (skeletonTmpl + yamlTitle/blockedBy funcs)
- internal/board/board.go, internal/board/board.md.tmpl (skeletonTmpl, filenames passed as data)
- internal/app/init.go, internal/app/readme.md.tmpl (renderReadme, BOARD.md via {{.Board}})
- tests/render_golden_test.go + tests/testdata/ (goldens captured from pre-swap code, verified after swap)
- task-system-spec.md §9 (embedded-template note)

Gates: golden tests byte-identical; full suite green (coverage 82.7%); gofmt clean; go vet clean; build ok.
