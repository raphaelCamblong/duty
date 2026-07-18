---
id: T-02
title: Task file domain package
status: done
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
- [x] Render→Parse round-trips; `SetStatus` changes exactly one line (byte-compare the
  rest); reports accumulate across two appends; gate counting handles 0/0, ticked,
  and mixed; slug rules covered — all as table-driven tests in `tests/`.
- [x] `go test ./tests/... -coverpkg=./internal/...` green; `gofmt -l .` empty;
  `go vet ./...` clean.

## Report

### 2026-07-09 — done

Files changed:
- `internal/task/task.go` — the whole package, pure (imports stdlib + yaml.v3
  only, zero filesystem): `Task` struct with yaml tags; `Parse` via
  `(?s)\A---\n(.*?)\n---\n` + `yaml.Unmarshal`; `Render` (frontmatter with
  `status: todo` + all six `##` headings, Gates empty so a fresh task counts
  0/0); `SetStatus` splicing only the first `(?m)^status: \S+` match (rejects
  values outside the four statuses); `AppendReport` (creates `## Report` once
  at EOF when missing, blank-line-separates accumulated blocks, pure append —
  never rewrites existing bytes); `CountGates` (`- [x]`/`- [ ]` prefixes under
  `## Gates`, stops at next `^## `); `Slugify` (ASCII lowercase/digits, runs of
  everything else collapse to one hyphen, no leading/trailing hyphen, ≤40 with
  post-truncation hyphen trim); status constants + `ValidStatus`.
- `tests/task_test.go` — `package tests`, black-box, table-driven `t.Run`
  subtests (64 runs): Render→Parse round-trips incl. titles needing YAML
  quoting (`: `, ` #`, quotes, unicode); `SetStatus` byte-compared against the
  full expected file (decoy `status:` line in body untouched, inline comment
  after the value survives) plus no-status-line and invalid-status errors;
  reports accumulate across two appends with the heading created exactly once;
  gate counting 0/0 (missing and empty section), all-ticked, mixed, stop at
  next section, pre-section and indented checkboxes ignored; slug table incl.
  40-char truncation edge; `ValidStatus` table; `Parse` error table.

Gate output tails:
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok github.com/raphaelCamblong/duty/tests 0.497s coverage: 90.3% of
  statements in ./internal/...`.
- `gofmt -l .` → empty; `go vet ./...` → clean;
  `go build -o bin/duty ./cmd/duty` → exit 0. golangci-lint not installed.

Deviations / interpretation calls (Scope left them open):
- "full §3 template … section skeleton, Gates as an empty checklist" rendered
  as headings-only sections without §3's placeholder prose: placeholder `- [ ]`
  gate lines would count as real unticked gates (wrong TUI readout on a fresh
  task), and empty sections are what the worker fills. Frontmatter carries no
  inline comments for the same reason (machine-owned region stays minimal).
- `Render` quotes the title (strconv.Quote, valid YAML double-quoted style)
  only when it is not a safe plain scalar, so `create "duty: x"` still
  round-trips; generation only — existing frontmatter is never re-serialized.
- `SetStatus` validates the new value with `ValidStatus` before splicing, so a
  typo can never write an unknown status into a file (transitions stay free).

Follow-ups deliberately left: none.
