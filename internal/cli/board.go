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

const boardUsage = "usage: duty board <name> [--title T]"

// runBoard creates the sub-board <name>/ under the current board: a skeleton
// board index (H1 = title, default the name) plus archive/, and appends the
// courtesy bullet to the parent's "## Boards" section. It refuses when the
// folder already exists.
func runBoard(f fsys.FS, cwd string, args []string) error {
	set := flag.NewFlagSet("board", flag.ContinueOnError)
	title := set.String("title", "", "board title (default: the name)")
	pos, err := positionals(set, args, boardUsage)
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

	parentDir, err := tree.CurrentBoard(f, cwd)
	if err != nil {
		return err
	}
	sub := filepath.Join(parentDir, name)
	if _, err := f.Stat(sub); err == nil {
		return fmt.Errorf("cannot create board: %s already exists", sub)
	}
	parentPath := filepath.Join(parentDir, names.BoardFile)
	parent, err := f.ReadFile(parentPath)
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

	if err := f.MkdirAll(filepath.Join(sub, names.ArchiveDir)); err != nil {
		return fmt.Errorf("create board: %w", err)
	}
	if err := f.WriteFile(filepath.Join(sub, names.BoardFile), board.Render(t)); err != nil {
		return err
	}
	return f.WriteFile(parentPath, withBullet)
}
