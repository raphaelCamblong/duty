package app

import (
	"github.com/raphaelCamblong/duty/internal/task"
)

// Gates returns the gate checklist of the task id resolves to, in file order.
// Archived ids are read-only and rejected.
func (a App) Gates(cwd, id string) ([]task.Gate, error) {
	_, path, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return nil, err
	}
	content, err := a.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return task.Gates(content), nil
}

// AddGates appends a gate per text, in order, to the task id resolves to, under
// the tree write lock in one write, creating the Gates section when absent.
func (a App) AddGates(cwd, id string, texts []string) error {
	return a.editGates(cwd, id, func(content []byte) ([]byte, error) {
		return task.AddGates(content, texts), nil
	})
}

// SetGate sets the n-th gate's (1-based) ticked state on the task id resolves
// to, under the tree write lock.
func (a App) SetGate(cwd, id string, n int, done bool) error {
	return a.editGates(cwd, id, func(content []byte) ([]byte, error) {
		return task.SetGate(content, n, done)
	})
}

// SetAllGates ticks or unticks every gate of the task id resolves to, under the
// tree write lock in one write.
func (a App) SetAllGates(cwd, id string, done bool) error {
	return a.editGates(cwd, id, func(content []byte) ([]byte, error) {
		return task.SetAllGates(content, done), nil
	})
}

// editGates applies edit to the file of the task id resolves to under the tree
// write lock, writing the result — the shared spine of the gate mutators.
func (a App) editGates(cwd, id string, edit func([]byte) ([]byte, error)) error {
	root, path, err := a.resolveOpenWithRoot(cwd, id)
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
	out, err := edit(content)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(path, out)
}
