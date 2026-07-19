package app

import (
	"bytes"
	"fmt"
	"io"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Section returns the named section's body (framing blanks trimmed) of the task
// id resolves to, erroring on no case-insensitive match; archived ids rejected.
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

// SetSection replaces the named section's body on the task id resolves to with
// reader's text (blank refused, a missing section created); reader is read after the id resolves.
func (a App) SetSection(cwd, id, name string, reader io.Reader) error {
	return a.editSection(cwd, id, "section", reader, func(content, payload []byte) ([]byte, error) {
		return task.ReplaceSection(content, name, payload)
	})
}

// SetSections replaces every "## " block read from reader on the task id resolves to
// in one write; blank or non-"## "-opening input is refused, reader read after the id.
func (a App) SetSections(cwd, id string, reader io.Reader) error {
	return a.editSection(cwd, id, "sections", reader, task.ReplaceSections)
}

// editSection is the shared read/lock/write spine of SetSection and
// SetSections.
func (a App) editSection(cwd, id, kind string, reader io.Reader, edit func(content, payload []byte) ([]byte, error)) error {
	root, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	payload, err := readNonBlank(reader, kind)
	if err != nil {
		return err
	}
	return a.lockedEdit(root, path, func(content []byte) ([]byte, error) {
		return edit(content, payload)
	})
}
