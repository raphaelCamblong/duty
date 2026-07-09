---
id: T-04
title: Tree discovery and id resolution
status: todo
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
- [ ] Tests build nested trees in `t.TempDir()`: walk-up from a deep cwd, topmost-root
  vs `duty.toml`-marked root, archived NN blocking reuse, archived id → read-only
  error, nested `duty.toml` → error.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
