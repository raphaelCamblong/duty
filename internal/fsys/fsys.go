// Package fsys is duty's filesystem port: an FS interface holding exactly the
// operations the system needs, plus an OS adapter over the real filesystem and
// a Mem adapter for fast in-memory tests. Every write is atomic.
package fsys

import "io/fs"

// FS is the filesystem port. Every filesystem touch in duty goes through it,
// so a reader never observes a half-written file: WriteFile is atomic.
type FS interface {
	// ReadFile returns the contents of the file at name.
	ReadFile(name string) ([]byte, error)
	// WriteFile writes data to name atomically, creating or replacing it.
	WriteFile(name string, data []byte) error
	// Rename moves the file at oldpath to newpath.
	Rename(oldpath, newpath string) error
	// Remove deletes the file at name.
	Remove(name string) error
	// MkdirAll creates the directory at path and every missing parent.
	MkdirAll(path string) error
	// ReadDir lists the directory at name, sorted by filename.
	ReadDir(name string) ([]fs.DirEntry, error)
	// Stat returns the FileInfo describing name.
	Stat(name string) (fs.FileInfo, error)
	// WalkDir walks the tree rooted at root in lexical order, calling fn for
	// every entry, honouring fs.SkipDir and fs.SkipAll.
	WalkDir(root string, fn fs.WalkDirFunc) error
}
