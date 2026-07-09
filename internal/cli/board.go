package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsutil"
	"github.com/raphaelCamblong/duty/internal/tree"
)

const boardUsage = "usage: duty board <name> [--title T]"

// runBoard creates the sub-board <name>/ under the current board: a skeleton
// BOARD.md (H1 = title, default the name) plus archive/, and appends the
// courtesy bullet to the parent's "## Boards" section. It refuses when the
// folder already exists.
func runBoard(cwd string, args []string) error {
	fs := flag.NewFlagSet("board", flag.ContinueOnError)
	title := fs.String("title", "", "board title (default: the name)")
	pos, err := positionals(fs, args, boardUsage)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New(boardUsage)
	}
	name := pos[0]
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid board name %q: must match [a-z0-9-]+", name)
	}

	parentDir, err := tree.CurrentBoard(cwd)
	if err != nil {
		return err
	}
	sub := filepath.Join(parentDir, name)
	if _, err := os.Stat(sub); err == nil {
		return fmt.Errorf("cannot create board: %s already exists", sub)
	}
	parentPath := filepath.Join(parentDir, boardFile)
	parent, err := os.ReadFile(parentPath)
	if err != nil {
		return err
	}
	t := *title
	if t == "" {
		t = name
	}
	withBullet, err := board.AddBoardBullet(parent, name, t)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(sub, "archive"), 0o755); err != nil {
		return fmt.Errorf("create board: %w", err)
	}
	if err := fsutil.WriteAtomic(filepath.Join(sub, boardFile), board.Render(t)); err != nil {
		return err
	}
	return fsutil.WriteAtomic(parentPath, withBullet)
}
