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
	root, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	text, err := readNonBlank(r, "section")
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := task.ReplaceSection(content, name, text)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(path, out)
}
