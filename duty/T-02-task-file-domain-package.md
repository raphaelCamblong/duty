---
id: T-02
title: Task file domain package
status: todo
blocked-by: [T-01]
---

# T-02 — Task file domain package

## Goal
`internal/task`: a pure (bytes in → bytes out, zero filesystem) model of the task file
format from spec §3.

## Read first
`task-system-spec.md` §3, §5, §9; `CLAUDE.md` architecture + code rules.

## Scope
- `Parse(content []byte) (Task, error)` — frontmatter via `\A---\n(.*?)\n---\n` +
  `yaml.v3` into `Task{ID, Title, Status, BlockedBy}`.
- `Render(id, title string, blockedBy []string) []byte` — the full §3 template
  (frontmatter + `##` section skeleton, Gates as an empty checklist).
- `SetStatus(content []byte, status string) ([]byte, error)` — targeted edit of the
  first `(?m)^status: \S+` match only; every other byte survives.
- `AppendReport(content, text []byte) []byte` — create `## Report` once, accumulate.
- `CountGates(content []byte) (done, total int)` — `- [x]` vs `- [ ]` lines under
  `## Gates`, stopping at the next `^## `.
- `Slugify(title string) string` — lowercase, non-alphanumerics → `-`, collapsed, ≤40.
- Status constants + `ValidStatus(s string) bool`.
- Only dependency: `gopkg.in/yaml.v3` (reads only — never re-serialize).

## Out of scope
Filesystem access, board files, id numbering (`tree` owns it).

## Gates
- [ ] Render→Parse round-trips; `SetStatus` changes exactly one line (byte-compare the
  rest); reports accumulate across two appends; gate counting handles 0/0, ticked,
  and mixed; slug rules covered — all as table-driven tests in `tests/`.
- [ ] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report
