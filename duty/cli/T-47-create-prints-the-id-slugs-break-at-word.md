---
id: T-47
title: Create prints the id, slugs break at words
status: todo
blocked-by: []
---

# T-47 — Create prints the id, slugs break at words

## Goal
`duty create task` answers with the id you need next, not just a path — and
long titles truncate into readable filenames.

## Read first
`internal/app/create.go` (returns the path today), `internal/cli/create.go`
(prints it), `task.Slugify` (hard cut at 40 chars mid-word — produced
`...structure-voice-screensho.md`), the tests that pin create's output.

## Scope
- `create task` stdout becomes one line: `<id>\t<path>` (id first, tab,
  absolute path). Same shape human and agent — it's already parseable; no
  separate --agent form needed. SANCTIONED BEHAVIOR CHANGE: update the tests
  that pin the old path-only output (assertions get stricter, not weaker).
  `create track` stays quiet (nothing to address it by).
- `task.Slugify`: still ≤40 chars, but truncate at the last word boundary
  (hyphen) that fits; only fall back to a hard cut when the first word alone
  exceeds the limit. Update the truncation test cases accordingly.
- `docs/cli.md`: the create section shows the new output line.

## Out of scope
Output changes to any other command; slug charset rules; renaming existing
task files.

## Gates
- [ ] `duty create task "x"` in a scratch tree prints `T-NN<TAB>/abs/path`;
  a 60-char multi-word title yields a slug cut at a word boundary ≤40.
- [ ] All tests green (`just check`); pinned-output tests updated, none
  weakened; docs updated.
