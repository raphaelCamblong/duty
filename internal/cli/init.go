package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
)

const initUsage = "usage: duty init [title]"

// readme is the agent-facing convention page duty init drops next to the
// root board index (spec §9): the command table, the lifecycle→command
// mapping, and what stays the worker's judgment.
const readme = `# duty/ — the task convention

One file = one task, small enough for one worker (human or agent) in one sitting.
[` + names.BoardFile + `](` + names.BoardFile + `) is the index: order top-to-bottom = build order. The task file is
the truth; the board is a projection the ` + "`duty`" + ` CLI keeps in sync.

## Task file

Frontmatter (` + "`id`, `title`, `status`, `blocked-by`" + `) + sections: ` + "`## Goal`" + `,
` + "`## Read first`" + `, ` + "`## Scope`" + ` (decisions are pre-made — don't re-decide),
` + "`## Out of scope`" + `, ` + "`## Gates`" + ` (a ` + "`- [ ]`" + ` checklist; all ticked = done),
` + "`## Report`" + ` (append, never overwrite).

Statuses: ` + "`todo | in-progress | done | blocked`" + `.

## Commands

| Command | Behavior |
|---|---|
| ` + "`duty create <title>`" + ` | New task in the current board (` + "`--slug`, `--blocked-by`, `--section`" + `). |
| ` + "`duty board <name>`" + ` | New sub-board under the current board (` + "`--title`" + `). |
| ` + "`duty status <id> <status>`" + ` | Set status in the task file AND its board row. |
| ` + "`duty link <id> <section>`" + ` | Move the board row under ` + "`## <section>`" + `. |
| ` + "`duty report <id>`" + ` | Append stdin under the task's ` + "`## Report`" + `. |
| ` + "`duty move <id> <board-path>`" + ` | Move a task to another board (path from the tree root). |
| ` + "`duty archive`" + ` | Move every ` + "`done`" + ` task into its board's ` + "`archive/`" + `. |
| ` + "`duty delete <id>`" + ` | Remove an open task (` + "`--force`" + ` for ` + "`done`" + `). |
| ` + "`duty list`" + ` | Open tasks from the files, with drift flags (` + "`--agent`" + ` for TSV). |
| ` + "`duty tui`" + ` | Live board viewer. |

## Lifecycle → command

1. Start → ` + "`duty status <id> in-progress`" + `.
2. Blocked (missing input, failed dep, unmade decision) → ` + "`duty status <id> blocked`" + `
   + pipe a report naming exactly what's missing (` + "`duty report <id>`" + `), then stop.
   Never guess past a blocker.
3. Working → tick gate checkboxes in the task file as they pass.
4. Done (all gates ticked) → ` + "`duty status <id> done`" + ` + pipe a report: files changed,
   gate output tails, deviations (with why), follow-ups deliberately left.
5. Respect ` + "`blocked-by`" + `: don't start a task whose dependencies aren't ` + "`done`" + `.
6. If a task turns out to be two, finish the stated scope and name the split in the
   report — don't expand.

## What stays your judgment

Filling Goal/Scope/Gates when authoring a task, ticking gates honestly, report prose,
and respecting ` + "`blocked-by`" + ` — the tooling checks none of it.
`

// runInit bootstraps a duty tree in cwd: duty/ with a skeleton board index
// (H1 = title, default "Board"), the convention readme, and archive/. It
// refuses to run inside an existing tree.
func runInit(f fsys.FS, cwd string, args []string) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	pos, err := positionals(set, args, initUsage)
	if err != nil {
		return err
	}
	if len(pos) > 1 {
		return errors.New(initUsage)
	}
	title := "Board"
	if len(pos) == 1 && pos[0] != "" {
		title = pos[0]
	}
	if dir, err := tree.CurrentBoard(f, cwd); err == nil {
		return fmt.Errorf("already inside a duty tree (%s)", dir)
	}
	dir := filepath.Join(cwd, names.TreeDir)
	if err := f.MkdirAll(filepath.Join(dir, names.ArchiveDir)); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	if err := f.WriteFile(filepath.Join(dir, names.BoardFile), board.Render(title)); err != nil {
		return err
	}
	return f.WriteFile(filepath.Join(dir, names.ReadmeFile), []byte(readme))
}
