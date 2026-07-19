package app

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// TaskSpec describes a task to create. Zero values default: an empty Slug is
// derived from Title, an empty Section is the default board section, and a nil
// Body renders the section skeleton instead of piped "## " markdown.
type TaskSpec struct {
	Title     string
	Slug      string
	Section   string
	BlockedBy []string
	Body      []byte
}

// CreateTask creates spec's task (status todo) in scope's board, returning its new id and path.
func (a App) CreateTask(scope Scope, spec TaskSpec) (id, path string, err error) {
	if spec.Slug != "" && !task.ValidSlug(spec.Slug) {
		return "", "", fmt.Errorf("invalid slug %q: want 1-40 chars of [a-z0-9-], no leading or trailing hyphen", spec.Slug)
	}
	if err := validateBody(spec.Body); err != nil {
		return "", "", err
	}
	boardDir, err := a.contextBoard(scope)
	if err != nil {
		return "", "", err
	}
	root, err := tree.FindRoot(a.fs, scope.Cwd)
	if err != nil {
		return "", "", err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return "", "", err
	}
	defer unlock()
	return a.createTaskLocked(root, boardDir, spec)
}

// validateBody rejects a piped body that is blank or does not open at a "## "
// heading; a nil body (no --body) passes untouched.
func validateBody(body []byte) error {
	if body == nil {
		return nil
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("empty body: pipe the body text on stdin")
	}
	return task.RequireOpensAtSection(body)
}

func (a App) createTaskLocked(root, boardDir string, spec TaskSpec) (id, path string, err error) {
	if err := a.validateBlockedBy(root, spec.BlockedBy); err != nil {
		return "", "", err
	}
	nn, err := tree.NextNN(a.fs, root)
	if err != nil {
		return "", "", err
	}
	if spec.Slug == "" {
		spec.Slug = task.Slugify(spec.Title)
	}
	if spec.Slug == "" {
		return "", "", fmt.Errorf("cannot derive a slug from %q, pass --slug", spec.Title)
	}
	if spec.Section == "" {
		spec.Section = board.DefaultSection
	}
	id = task.FormatID(nn)
	path, err = a.writeTask(boardDir, id, spec)
	return id, path, err
}

// validateBlockedBy treats an archived dependency as a legal blocked-by id.
func (a App) validateBlockedBy(root string, blockedBy []string) error {
	for _, dep := range blockedBy {
		if _, err := tree.ResolveTask(a.fs, root, dep); err != nil && !errors.Is(err, tree.ErrArchived) {
			return fmt.Errorf("blocked-by: %w", err)
		}
	}
	return nil
}

// writeTask computes the board row before writing either file, so a failure
// leaves neither written.
func (a App) writeTask(boardDir, id string, spec TaskSpec) (string, error) {
	filename := id + "-" + spec.Slug + ".md"
	boardPath := boardIndexPath(boardDir)
	content, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return "", err
	}
	withRow, err := board.AddRow(content, spec.Section, board.Row{ID: id, File: filename, Title: spec.Title, Status: task.StatusTodo})
	if err != nil {
		return "", err
	}
	taskPath := filepath.Join(boardDir, filename)
	if err := a.fs.WriteFile(taskPath, renderTask(id, spec.Title, spec.BlockedBy, spec.Body)); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(boardPath, withRow); err != nil {
		return "", err
	}
	return taskPath, nil
}

func renderTask(id, title string, blockedBy []string, body []byte) []byte {
	if body == nil {
		return task.Render(id, title, blockedBy)
	}
	return task.RenderWithBody(id, title, blockedBy, body)
}
