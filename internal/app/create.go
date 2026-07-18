package app

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// CreateTask creates a task in the board in — a root-relative track path, or
// the board containing cwd when empty — and returns the new task's id and
// file path: every blocked-by id is validated against the whole tree, the
// task is numbered tree-wide, and the task file and its board row (status
// todo) are written in one use-case. An empty slug is derived from the title;
// an empty section means the default one. A non-nil body is the one-shot
// markdown read from stdin (## sections verbatim below a generated H1); nil
// renders the section skeleton instead.
func (a App) CreateTask(cwd, title, slug, section, in string, blockedBy []string, body io.Reader) (id, path string, err error) {
	if slug != "" && !task.ValidSlug(slug) {
		return "", "", fmt.Errorf("invalid slug %q: want 1-40 chars of [a-z0-9-], no leading or trailing hyphen", slug)
	}
	bodyBytes, err := readTaskBody(body)
	if err != nil {
		return "", "", err
	}
	boardDir, err := a.contextBoard(cwd, in)
	if err != nil {
		return "", "", err
	}
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", "", err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return "", "", err
	}
	defer unlock()
	return a.createTaskLocked(root, boardDir, title, slug, section, blockedBy, bodyBytes)
}

// readTaskBody reads an optional one-shot task body: a nil reader (no --body)
// yields no body, else stdin is required non-blank and must open at a "## "
// heading — the frontmatter and H1 are generated, never piped.
func readTaskBody(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	text, err := readNonBlank(r, "body")
	if err != nil {
		return nil, err
	}
	if err := task.RequireOpensAtSection(text); err != nil {
		return nil, err
	}
	return text, nil
}

// createTaskLocked validates dependencies, numbers the task tree-wide, and
// writes the task file and its board row. It must run under the tree lock.
func (a App) createTaskLocked(root, boardDir, title, slug, section string, blockedBy []string, body []byte) (id, path string, err error) {
	if err := a.validateBlockedBy(root, blockedBy); err != nil {
		return "", "", err
	}
	nn, err := tree.NextNN(a.fs, root)
	if err != nil {
		return "", "", err
	}
	if slug == "" {
		slug = task.Slugify(title)
	}
	if slug == "" {
		return "", "", fmt.Errorf("cannot derive a slug from %q, pass --slug", title)
	}
	if section == "" {
		section = board.DefaultSection
	}
	id = task.FormatID(nn)
	path, err = a.writeTask(boardDir, id, slug, title, section, blockedBy, body)
	return id, path, err
}

// validateBlockedBy checks every dependency id resolves somewhere in the
// tree; archived dependencies are legal.
func (a App) validateBlockedBy(root string, blockedBy []string) error {
	for _, dep := range blockedBy {
		if _, err := tree.ResolveTask(a.fs, root, dep); err != nil && !errors.Is(err, tree.ErrArchived) {
			return fmt.Errorf("blocked-by: %w", err)
		}
	}
	return nil
}

// writeTask renders the task file and appends its board row (status todo),
// both contents computed before either write, and returns the new file's path.
func (a App) writeTask(boardDir, id, slug, title, section string, blockedBy []string, body []byte) (string, error) {
	filename := id + "-" + slug + ".md"
	boardPath := boardIndexPath(boardDir)
	content, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return "", err
	}
	withRow, err := board.AddRow(content, section, id, filename, title, task.StatusTodo)
	if err != nil {
		return "", err
	}
	taskPath := filepath.Join(boardDir, filename)
	if err := a.fs.WriteFile(taskPath, renderTask(id, title, blockedBy, body)); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(boardPath, withRow); err != nil {
		return "", err
	}
	return taskPath, nil
}

// renderTask renders a new task file: the one-shot body below a generated H1
// when body is non-nil, else the full section skeleton.
func renderTask(id, title string, blockedBy []string, body []byte) []byte {
	if body == nil {
		return task.Render(id, title, blockedBy)
	}
	return task.RenderWithBody(id, title, blockedBy, body)
}
