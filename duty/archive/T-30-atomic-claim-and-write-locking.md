---
id: T-30
title: Atomic claim and write locking
status: done
blocked-by: [T-29]
---

# T-30 — Atomic claim and write locking

## Goal
Parallel agents are safe: `duty get next --claim` hands each caller a distinct
task atomically, and every mutating command serializes on a tree-wide lock.

## Read first
`task-system-spec.md` §6 (`get next`) and §10 ("No locking" — this task
retires that bullet); `internal/fsys` (the port both adapters implement);
`internal/app/get.go` (`GetNext`), `status.go`.

## Scope
- **Lock primitive in the port:** `fsys.FS` gains
  `Lock(path string) (unlock func(), err error)` — `OS` via
  `github.com/gofrs/flock` (the one new dep; cross-platform flock), blocking
  acquire with a ~5s timeout → error `tree is locked`; `Mem` via a per-path
  in-process mutex. Lock file: `<root>/.duty.lock` — name lives in
  `internal/names`, added to the repo `.gitignore` and to `init`'s generated
  tree behavior (created on demand, never committed).
- **Every mutating app method takes the lock** for its duration: `CreateTask`,
  `CreateTrack`, `SetStatus`, `Report`, `Move`, `Archive`, `Delete` (not
  `Init` — no tree yet; not reads). One guard helper in `app`, acquired at
  entry, released on return.
- **`duty get next --claim`:** under the same lock: compute the next
  actionable task, set it `in-progress` (task file + board row, the normal
  sync write), then print it exactly like `get next` (status shows
  `in-progress` — the truthful post-claim state; same `--agent` TSV shape).
  Losers of the race transparently receive the following task. Nothing
  actionable → empty output, exit 0, no lock-file side effects.
- **Claim conflict guard:** `duty status <id> in-progress` on a task already
  `in-progress` errors (`T-x is already in-progress — someone claimed it; use
  --force to take it over`) unless `--force`. Only that transition; all other
  transitions stay free (flat setter per spec §3).
- Docs in the same change: spec §6 (`--claim` on the get next row, `--force`
  on status) and §10 (retire "No locking" — describe the flock design and why
  claim makes parallel agents safe); README agents paragraph gains one line
  (`duty get next --claim` for parallel agents); template + golden if touched.
- Tests: concurrency test — N goroutines run `get next --claim` via `cli.Run`
  on a tree with N actionable tasks → N distinct ids claimed, board and files
  consistent after; claim-empty case; status in-progress refusal + `--force`;
  lock timeout error path (Mem).

## Out of scope
Per-task lock granularity; lease/heartbeat/stale-claim recovery (a crashed
agent leaves in-progress — `--force` is the manual recovery, note it in spec
§10); locking reads; TUI.

## Gates
- [x] Concurrency test green under `-race`: N parallel claims → N distinct
  tasks, tree hash-consistent (every file/row pair in sync).
- [x] `status <id> in-progress` refusal + `--force` override covered by tests;
  roundtrip and all existing tests still green (transition tightening is the
  ONLY sanctioned behavior change).
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`
  and `-race`); `golangci-lint run` clean; `gofumpt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [x] Spec §6 + §10, README, `.gitignore`, `internal/names` all updated
  together.

## Report

Delivered atomic claim + tree-wide write locking.

fsys port: FS gains Lock(path) (unlock, err). OS adapter (internal/fsys/os.go)
uses github.com/gofrs/flock — blocking acquire with a 5s timeout, "tree is
locked" on timeout. Mem adapter (mem.go) uses a per-path buffered-channel lock
with a configurable LockTimeout (default 5s) for the same error path. Shared
sentinel message errLocked lives in fsys.go. Lock file <root>/.duty.lock named
in internal/names, gitignored.

app: one guard (App.lock → fs.Lock(root/.duty.lock)) wraps every mutating
use-case for its duration — CreateTask, CreateTrack, SetStatus, Report, Move,
Archive, Delete (Init and reads excluded). Read-modify-write bodies extracted
into *Locked helpers to stay under funlen 35. get next --claim (app/get.go
claim) peeks unlocked, then under the lock re-scans (authoritative) and marks
the first actionable task in-progress, returning it with the truthful status;
an empty claim never touches the lock file. SetStatus gained a force param;
guardClaim refuses in-progress→in-progress without --force, all other
transitions stay free.

cli: status --force, get next --claim, get next usage/example updated.

Gate tails: gofmt/gofumpt clean; go vet clean; golangci-lint "0 issues";
go test ./tests/... -coverpkg=./internal/... → ok, coverage 85.7%;
go test -race → ok (incl. TestClaimParallel: 8 goroutines → 8 distinct tasks,
board+files in sync). New tests: tests/cli_claim_test.go (parallel claims,
single claim, empty-claim no-lock-file, status --force guard),
tests/fsys_test.go TestLock (OS mutual exclusion, Mem timeout, re-acquire).

Docs updated together: spec §6 (status --force row, get next --claim row, lock
note in intro) + §10 (retired "No locking", describes the flock design, notes
--force as stale-claim recovery); root README agents paragraph; agent template
readme.md.tmpl + regenerated golden tests/testdata/readme.md; .gitignore;
internal/names.LockFile.

Design note: the .duty.lock artifact is created on demand and gitignored, so
the two byte-identity test fingerprints (hashTree, snapshotTree) now skip it —
the task-content round-trip invariant is unchanged. init does not pre-create a
per-tree .gitignore (YAGNI; the file is gitignored in this repo and lives only
transiently). No follow-ups; scope complete.

Simplify pass (T-30 quality bar): extracted App.lockTree(cwd) for the FindRoot->lock->defer prelude (archive, create-track); reworked Move to resolve the id before locking so the write lock covers only the board read-modify-write like the sibling mutators, dropping a duplicate FindRoot walk; deleted the now-unused resolveOpen (inlined into moveRowInBoard). Skipped two findings: moving the "tree is locked" wording up out of the fsys port (would require editing the frozen fsys_test.go assertion) and merging the per-adapter ~5s timeout constants (per-adapter implementation detail, not part of the port contract). Suite green, race clean, golangci-lint 0 issues, gofumpt clean.
