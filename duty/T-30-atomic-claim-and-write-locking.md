---
id: T-30
title: Atomic claim and write locking
status: todo
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
- [ ] Concurrency test green under `-race`: N parallel claims → N distinct
  tasks, tree hash-consistent (every file/row pair in sync).
- [ ] `status <id> in-progress` refusal + `--force` override covered by tests;
  roundtrip and all existing tests still green (transition tightening is the
  ONLY sanctioned behavior change).
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`
  and `-race`); `golangci-lint run` clean; `gofumpt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [ ] Spec §6 + §10, README, `.gitignore`, `internal/names` all updated
  together.

## Report
