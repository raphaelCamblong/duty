---
id: T-64
title: APIs that explain themselves
status: in-progress
blocked-by: []
---

# T-64 — APIs that explain themselves

## Goal
The comment-heavy app contracts get refactored until their comments become
unnecessary: the API says what the prose currently apologizes for.

## Read first
CLAUDE.md's new rule (a usage-rule comment is a design bug — refactor until
unnecessary). The flagged evidence: internal/app/create.go (CreateTask's
8-line comment documenting three implicit modes), the "(cwd, in)" duality
re-explained verbatim on every board-scoped method, "must run under the tree
lock" repeated per *Locked helper, list.go's drift() whose comment exists
because its signature is awkward.

## Scope
1. **One Scope type for the (cwd, in) duality.** `app.Scope{Cwd, In string}`
   with the rule documented ONCE on the type ("In: root-relative track path;
   empty = the board containing Cwd"). Every board-scoped method
   (CreateTask, CreateTrack, List, GetTasks-path, GetTracks, GetNext,
   Archive, watch's snapshot scoping — find them all) takes Scope; their
   per-method restatements of the rule are deleted. cli builds Scope at the
   edge.
2. **TaskSpec owns its defaults.** Zero-value semantics move to the struct's
   field docs, stated once (Slug empty → derived from Title; Section empty →
   the default section). CreateTask's doc shrinks to one line.
3. **Body becomes data, not a reader mode.** TaskSpec gains `Body []byte`
   (nil = render the skeleton — a one-line field doc); the CLI reads stdin at
   the edge and validates happen in app as today (RequireOpensAtSection).
   Error-precedence guard: create already reads the body BEFORE resolving
   the board, so pre-reading in cli preserves observable error ordering.
   Report is explicitly OUT of scope — its reader-based ordering (id
   resolves before stdin is consumed) is a deliberate, test-adjacent
   contract; leave it and its one-line comment alone.
4. **The *Locked convention documented once.** Delete every "must run under
   the tree lock" comment; one sentence at App.lock (or the package doc)
   states: helpers with the Locked suffix require the tree lock.
5. **list.go drift() redesign.** Take `(index []byte, filename, fileStatus
   string)` and do the FindRow/RowStatus lookup inside (or return a tiny
   struct) so the coupled-params comment collapses; boardRows/boardOrder
   comments shrink to their irreducible one-liners (the nextInBoard
   mirroring contract stays, one line).
6. **Sweep app/ for the same pattern**: any remaining multi-line function
   comment gets the same test — can the API be made to say it? Refactor
   where yes; where the prose is a genuine irreducible contract, compress to
   one line and note it in the report.

## Out of scope
Report's io.Reader (deliberate ordering contract); cli/tui/domain packages
beyond mechanical call-site updates; behavior changes of any kind; new
features.

## Gates
- [ ] CreateTask's doc comment is at most 1 line; no board-scoped method
  re-documents the in-or-cwd rule (it lives once, on Scope); zero "must run
  under the tree lock" comments remain (one convention sentence exists).
- [ ] grep: no function comment in internal/app exceeds 2 lines except ones
  the report justifies as irreducible domain contracts (each named).
- [ ] Full suite green with only mechanical call-syntax test updates
  (listed, no assertion weakened); observable error ordering unchanged
  (create's body-before-board precedence preserved, test-verified).
- [ ] just check green; param-count scan does not regress (Scope should
  DROP several functions below 5 params — report before/after).
