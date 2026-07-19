---
id: T-62
title: Prune redundant comments across internal/
status: done
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
- [x] `just check` green (build, vet, gofumpt, golangci-lint 0 issues, full
  suite) with ZERO test file edits.
- [x] Spot check: internal/tui/entry.go and keys.go read leanly (report
  before/after comment-line counts for these two specifically, since they
  were the flagged examples).
- [x] Report totals: comments removed / kept-as-genuine per package group.

## Report

### 2026-07-19 18:59 — done

Verified `just check` green across all six T-62 sub-commits (gofumpt clean,
go vet clean, golangci-lint 0 issues, full test suite passing at 87.9%
coverage of internal/), zero test file edits. No cross-group integration
break found; no fix commit needed.

Comment-line totals (gross removed / added-back during rewording / net
removed / genuine comments kept, counted across internal/+cmd/, all six
commits diffed against 10c2cc7, the pre-T-62 baseline):

  humanize+fetch+watch      removed 26  added 10  net  16  kept  27
  fsys+tree+config+names+cmd removed 63  added  6  net  57  kept  98
  cli                        removed 162 added 24  net 138  kept  29
  task+board                 removed 91  added  0  net  91  kept 158
  app                        removed 128 added 50  net  78  kept 223
  tui                        removed 207 added  1  net 206  kept 266
  -----------------------------------------------------------------
  TOTAL                      removed 677 added 91  net 586  kept 801

No group was already clean — every one of the six had comments to prune,
ranging from 26 (humanize+fetch+watch, already terse) to 207 (tui, the
heaviest offender) gross comment lines deleted.

Flagged spot-check files (internal/tui, commit 3b4ed84):
  entry.go: 64 comment lines before -> 26 after
  keys.go:   5 comment lines before ->  0 after

Both read leanly now: entry.go keeps only the handful of comments that
state a real constraint; keys.go has none left, all of them were pure
restatements of the binding name.
