---
id: T-68
title: "APIs that explain themselves: everywhere load-bearing"
status: todo
blocked-by: []
---

# T-68 — APIs that explain themselves: everywhere load-bearing

## Goal
T-64's iteration — refactor until usage-rule comments become unnecessary —
applied to the remaining load-bearing packages: task, board, tree, cli, tui.

## Read first
CLAUDE.md's comment-escalation rule; T-64's pattern (types for dualities,
zero-value docs on structs once, modes as data, conventions stated once);
the multi-line comments surviving in these packages as the work list.

## Scope
- Per package (task+board, tree, cli, tui): find every function comment over
  ~2 lines; apply the "can the API say it?" test — refactor where yes
  (named types, explicit fields, single documented defaults), compress to
  one justified line where it is a genuine irreducible contract (byte-format
  guarantees and cross-function invariants qualify; usage rules do not).
- Behavior frozen; mechanical test-call updates only where signatures
  change, listed.

## Out of scope
app/ (done in T-64); the representation redesign's territory if T-67 landed
first (don't re-churn its fresh code — judge it as-is); fsys/config/names/
humanize/fetch/watch (already lean).

## Gates
- [ ] No function comment over 2 lines in the four package groups except
  named, justified irreducible contracts.
- [ ] just check green; suite green; no weakened assertions.
