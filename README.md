# duty

A file-based task system: every task is a markdown file, boards are nested folders of
them, and one Go binary keeps the files and their board indexes in sync. Boards nest by
convention — any folder with a `BOARD.md` is a board, recursively — so the filesystem is
the registry and structure can't drift. There is no database and no daemon: tasks are
plain text, greppable and diffable, kept in git next to the code they describe. `duty`
ships a one-shot CLI and a live TUI over the same files, and it's built agent-first —
every command is quiet on success, exits non-zero with a one-line reason on failure, and
speaks a stable TSV so an LLM can drive the whole lifecycle without parsing prose.

## A session

```console
$ duty init "Payments"        # scaffold ./duty with a root board
$ cd duty
$ duty create track api       # a track is just a folder with its own board
$ duty create task "Design the charge endpoint"
~/payments/duty/T-01-design-the-charge-endpoint.md
$ duty create task "Handle refunds" --blocked-by T-01
~/payments/duty/T-02-handle-refunds.md
$ duty get next               # first todo whose blocked-by are all done
id:         T-01
title:      Design the charge endpoint
status:     todo
track:      .
blocked-by: none
gates:      0/0
path:       ~/payments/duty/T-01-design-the-charge-endpoint.md
$ duty status T-01 in-progress
# ... tick the gate checkboxes in the task file as they pass ...
$ duty status T-01 done
$ duty archive                # every done task moves into its board's archive/
```

Each state change writes the task file's frontmatter *and* its board row in one command —
that sync invariant is the whole point. `duty --help` opens with this lifecycle and every
command carries a copy-pasteable example.

## Install

Until the first tagged release the packaged channels below don't exist yet; build from
source in the meantime.

```sh
# From source (works today):
git clone https://github.com/raphaelCamblong/duty
cd duty && go build -o bin/duty ./cmd/duty

# Homebrew tap (after v0.1.0):
brew install raphaelCamblong/tap/duty

# Go toolchain (after v0.1.0):
go install github.com/raphaelCamblong/duty/cmd/duty@latest
```

Prebuilt binaries for macOS, Linux, and Windows will be attached to each
[GitHub Release](https://github.com/raphaelCamblong/duty/releases) (after v0.1.0). `apt`
and `rpm` repositories are deliberately left for later.

## The TUI

`duty tui` launches a read-only, live board viewer: a master-detail layout — the tree of
tracks and tasks browses full-width on the left, and opening a task (`enter`) splits in a
glamour-rendered preview of its file on the right. The header shows a breadcrumb of the
current track plus a status-distribution bar for its subtree; `/` opens a fuzzy filter,
`j/k` and the mouse navigate (rows and breadcrumb segments are clickable), and `e` opens
the selected task in `$EDITOR`. It never writes: an fsnotify watcher re-scans the tree on
any change, so edits from the CLI or your editor appear within a blink. The files are the
truth; when a board row disagrees with a task file, the TUI flags the drift rather than
hiding it.

## For agents

The CLI is a one-shot contract, not a REPL: mutating commands are silent on success and
exit non-zero with a single lowercase stderr line on any problem, so exit codes *are* the
API. Reading commands take `--agent` for stable, tab-separated output with a fixed field
order — no color, no padding, cheap to `cut`/`awk` and cheap in tokens:

```sh
duty get next --agent     # id  track  status  title  gates-done  gates-total  blocked-by  path
duty get tasks --agent    # id  board  status  title  drift   (one record per open task)
```

`get next` returns the first actionable task (a `todo` whose `blocked-by` are all done)
and prints nothing when there's nothing to do. The task-file convention an agent fills in
— goal, scope, gates, reports — is documented in [duty/README.md](duty/README.md); the
full behavioral contract lives in [task-system-spec.md](task-system-spec.md).

## Development

```sh
go build -o bin/duty ./cmd/duty                        # build (bin/ is gitignored)
go test ./tests/... -coverpkg=./internal/...           # test
```

Architecture, layering, and code rules are in [CLAUDE.md](CLAUDE.md); the spec is the
source of truth for behavior.
