package app

import (
	"bytes"
	"fmt"
	"io"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Section returns the body of the named section of the task id resolves to,
// trimmed of its framing blank lines. It errors when no section matches the
// name (case-insensitively). Archived ids are read-only and rejected.
func (a App) Section(cwd, id, name string) (string, error) {
	_, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return "", err
	}
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	body, ok := task.Section(content, name)
	if !ok {
		return "", fmt.Errorf("no section %q in %s", name, id)
	}
	return string(bytes.TrimSpace(body)), nil
}

// SetSection replaces the named section's body of the task id resolves to with
// the text read from r, under the tree write lock. Empty (blank) input is
// refused; the heading line and every byte outside the section survive, and a
// missing section is created. r is read only after the id resolves.
func (a App) SetSection(cwd, id, name string, r io.Reader) error {
	return a.editSection(cwd, id, "section", r, func(content, payload []byte) ([]byte, error) {
		return task.ReplaceSection(content, name, payload)
	})
}

// SetSections replaces every "## <name>" block read from r on the task id
// resolves to, under the tree write lock in one file write: each named section
// is replaced (a missing one created, like SetSection), with every byte outside
// them surviving. Empty (blank) input, or input not opening at a "## " heading,
// is refused; r is read only after the id resolves.
func (a App) SetSections(cwd, id string, r io.Reader) error {
	return a.editSection(cwd, id, "sections", r, task.ReplaceSections)
}

// editSection is the shared read/lock/write spine of SetSection and
// SetSections.
func (a App) editSection(cwd, id, kind string, r io.Reader, edit func(content, payload []byte) ([]byte, error)) error {
	root, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	payload, err := readNonBlank(r, kind)
	if err != nil {
		return err
	}
	return a.lockedEdit(root, path, func(content []byte) ([]byte, error) {
		return edit(content, payload)
	})
}
