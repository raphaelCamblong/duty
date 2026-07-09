package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// CreateBoard creates the sub-board name/ under the board containing cwd: a
// skeleton board index (H1 = title, default the name) plus archive/, and the
// courtesy bullet appended to the parent's "## Boards" section. It refuses
// when the folder already exists.
func (a App) CreateBoard(cwd, name, title string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid board name %q: must match [a-z0-9-]+", name)
	}

	parentDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return err
	}
	sub := filepath.Join(parentDir, name)
	if _, err := a.fs.Stat(sub); err == nil {
		return fmt.Errorf("cannot create board: %s already exists", sub)
	}
	parentPath := filepath.Join(parentDir, names.BoardFile)
	parent, err := a.fs.ReadFile(parentPath)
	if err != nil {
		return err
	}
	if title == "" {
		title = name
	}
	withBullet, err := board.AddBoardBullet(parent, name, title)
	if err != nil {
		return err
	}

	if err := a.fs.MkdirAll(filepath.Join(sub, names.ArchiveDir)); err != nil {
		return fmt.Errorf("create board: %w", err)
	}
	if err := a.fs.WriteFile(filepath.Join(sub, names.BoardFile), board.Render(title)); err != nil {
		return err
	}
	return a.fs.WriteFile(parentPath, withBullet)
}
