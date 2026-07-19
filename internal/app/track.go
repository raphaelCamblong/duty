package app

import (
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
)

// CreateTrack creates the track name/ under scope's board (skeleton index,
// archive/, a parent Boards bullet) and returns its path, refusing an existing folder.
func (a App) CreateTrack(scope Scope, name, title string) (string, error) {
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("invalid track name %q: must match [a-z0-9-]+", name)
	}
	parentDir, err := a.contextBoard(scope)
	if err != nil {
		return "", err
	}
	unlock, err := a.lockTree(scope.Cwd)
	if err != nil {
		return "", err
	}
	defer unlock()
	return a.createTrackLocked(parentDir, name, title)
}

func (a App) createTrackLocked(parentDir, name, title string) (string, error) {
	sub := filepath.Join(parentDir, name)
	if _, err := a.fs.Stat(sub); err == nil {
		return "", fmt.Errorf("cannot create track: %s already exists", sub)
	}
	parentPath := boardIndexPath(parentDir)
	parent, err := a.fs.ReadFile(parentPath)
	if err != nil {
		return "", err
	}
	if title == "" {
		title = name
	}
	withBullet, err := board.AddBoardBullet(parent, name, title)
	if err != nil {
		return "", err
	}
	if err := a.fs.MkdirAll(filepath.Join(sub, names.ArchiveDir)); err != nil {
		return "", fmt.Errorf("create track: %w", err)
	}
	if err := a.fs.WriteFile(boardIndexPath(sub), board.Render(title)); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(parentPath, withBullet); err != nil {
		return "", err
	}
	return sub, nil
}
