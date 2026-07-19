---
id: T-68
title: "APIs that explain themselves: everywhere load-bearing"
status: done
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
- [x] No function comment over 2 lines in the four package groups except
  named, justified irreducible contracts.
- [x] just check green; suite green; no weakened assertions.

## Report

### 2026-07-19 22:08 — done

The T-64 iteration, applied to everything load-bearing that remained:
task+board+tree, cli, and tui (app was T-64's; T-67's fresh loader code
was judged as written). Comment lines 494 to 367 across the four groups —
but the point was the dissolutions, not the count:

- task: maxSlugLen names the 40 three functions hard-coded;
  RenderWithBody's precondition prose replaced by the named guard already
  at its call site.
- board: reorderAdjacent's 0/1 magic-value note deleted — the duality was
  already split into ReorderBefore/ReorderAfter at the public API.
- cli: readBody's nil-means-a-mode comment deleted (the signature and
  TaskSpec.Body's documented zero value say it).
- tui: renderPreview(reset bool) — a textbook parameter duality — became
  renderPreview() with the one resetting caller doing its own scroll-to-
  top; spinner lifecycle prose collapsed to its one cross-function
  contract line.

What survives is exactly the sanctioned category, one line each:
byte-format guarantees (line-surgical setters, exact-inverse split/join,
never-rewrites-content appends), cross-function invariants (prune never
removes the default section, trackRightWidth alignment, arm/onSpinnerTick
lifecycle), and terminal-quirk contracts (the OSC query that caused the
v0.4.0 freeze, bubbletea v2 lifecycle notes). Zero behavior change:
round-trip, salted-board and both byte-exact theme goldens pass
untouched; one unexported signature changed (renderPreview), three call
sites, no test fallout. just check green, coverage 88.7%.
