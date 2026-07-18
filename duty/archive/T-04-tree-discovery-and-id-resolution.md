---
id: T-04
title: Tree discovery and id resolution
status: done
blocked-by: [T-01]
---

# T-04 — Tree discovery and id resolution

## Goal
`internal/tree`: locate the tree, its boards, its tasks, and the next id on a real
filesystem, per spec §2–§3 and the §7 root rules.

## Read first
`task-system-spec.md` §2, §3, §7; `CLAUDE.md`.

## Scope
- `FindRoot(cwd)` — walk up to the nearest `BOARD.md`, then continue to the topmost
  `BOARD.md`; a dir holding `duty.toml` marks the root explicitly. Outside a tree:
  fall back to `./duty/` if it exists, else a one-line error.
- `CurrentBoard(cwd)` — nearest ancestor with `BOARD.md`.
- `Boards(root)` — `filepath.WalkDir` collecting dirs containing `BOARD.md`, skipping
  `archive/`. Errors if a `duty.toml` exists below the root (second-root, spec §7).
- `ResolveTask(root, id)` — walk matching `<id>-*.md`; an archived match is a distinct
  "read-only" error naming the id.
- `NextNN(root)` — max NN across ALL task filenames, open + archived, every board,
  plus one; zero-padded to two digits minimum.

## Out of scope
Parsing file contents (`task`/`board` own that), config values, any mutation.

## Gates
- [x] Tests build nested trees in `t.TempDir()`: walk-up from a deep cwd, topmost-root
  vs `duty.toml`-marked root, archived NN blocking reuse, archived id → read-only
  error, nested `duty.toml` → error.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

### 2026-07-09 — done

Files changed:
- `internal/tree/tree.go` (new) — `FindRoot`, `CurrentBoard`, `Boards`,
  `ResolveTask`, `NextNN`, sentinel `ErrArchived`; helpers `nearestBoard`,
  `fallbackTree`, `hasFile`, `underArchive`. Read-only: reads directory
  structure and filenames only, never file contents.
- `tests/tree_test.go` (new) — table-driven black-box tests, all fixtures via
  `t.TempDir()` through a local `buildTree` helper (kept in-file per parallel
  worktree discipline, not in a shared helpers file). Covers every scenario in
  gate 1 plus fallback-to-`./duty`, contiguity stop, non-prefix id matching,
  and NN growth past two digits.

Gate output tails:
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok github.com/raphaelCamblong/duty/tests 0.394s coverage: 87.3% of statements in ./internal/...`
- `gofmt -l .` → empty; `go vet ./...` → clean;
  `go build -o bin/duty ./cmd/duty` → exit 0. `golangci-lint` not installed.

Interpretation decisions (within spec, noting for reviewers):
- `FindRoot` ascends only while the parent also holds `BOARD.md` (boards nest
  contiguously per §2), so a stray unrelated `BOARD.md` above a gap is never
  adopted as root; `duty.toml` on a board in the chain stops the ascent there
  even if boards exist above (§7 explicit-root rule).
- The `./duty/` fallback (spec §2 applies it to current-board resolution, so
  both `FindRoot` and `CurrentBoard` use it) requires only that `./duty` exists
  as a directory; missing `BOARD.md` inside surfaces downstream.
- `Boards` errors on a `duty.toml` in ANY directory below root, board or not
  (scope: "a duty.toml exists below the root"); `archive/` is skipped before
  the check, so a toml buried in an archive is not scanned.
- `ResolveTask` matches `<id>-` prefix + `.md` suffix, so `T-1` never resolves
  `T-10-*.md`; the archived error wraps `ErrArchived` (branchable via
  `errors.Is`) and names the id.

No deviations from the spec; no follow-ups left.
