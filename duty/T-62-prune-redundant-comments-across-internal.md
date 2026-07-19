---
id: T-62
title: Prune redundant comments across internal/
status: todo
blocked-by: []
---

# T-62 — Prune redundant comments across internal/

## Goal
Comments across the codebase stop paraphrasing signatures. Most existing
one-line godocs and every multi-line function/struct comment get deleted;
only genuine non-obvious constraints survive.

## Read first
CLAUDE.md's rewritten comment rule (comments are the exception, not the
default — delete anything that just restates the code below it; keep exactly
one terse line only when it adds real information). User-flagged examples of
the smell: internal/tui/entry.go, internal/tui/keys.go — read them as the
template for what "too much" looks like.

## Scope
- Sweep every .go file under internal/ and cmd/ (package by package).
  For each comment above a func/type/const/var: keep it only if it states a
  constraint, invariant, or gotcha the reader cannot get from the name +
  signature + body in a few seconds. Otherwise delete it entirely — do NOT
  shrink it to a shorter paraphrase, remove it.
- Struct field comments: same rule — delete field-by-field restatements
  ("nameW is the name column width"), keep only a field whose meaning is
  genuinely non-obvious from its name and type.
- Zero behavior change. Zero test edits (this only touches comments).
- Package doc comments (the one atop each package's main file) may shrink
  but each package keeps at least a one-line identification.

## Out of scope
Any code/logic change; the many-argument-function refactor (separate task);
docs/*.md content; test files.

## Gates
- [ ] `just check` green (build, vet, gofumpt, golangci-lint 0 issues, full
  suite) with ZERO test file edits.
- [ ] Spot check: internal/tui/entry.go and keys.go read leanly (report
  before/after comment-line counts for these two specifically, since they
  were the flagged examples).
- [ ] Report totals: comments removed / kept-as-genuine per package group.
