package fsys

import (
	"io/fs"
	"path/filepath"
	"sort"
	"time"
)

// Mem is a map-backed in-memory FS for fast tests. Files carry mode 0644,
// directories 0755; a file's parent directory must exist before it is written.
type Mem struct {
	files map[string][]byte
	dirs  map[string]bool
}

var _ FS = (*Mem)(nil)

// NewMem returns an empty in-memory filesystem holding only the root "/".
func NewMem() *Mem {
	return &Mem{
		files: map[string][]byte{},
		dirs:  map[string]bool{filepath.Clean("/"): true},
	}
}

// ReadFile returns a copy of the contents of the file at name.
func (m *Mem) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[filepath.Clean(name)]
	if !ok {
		return nil, notExist("open", name)
	}
	return append([]byte(nil), data...), nil
}

// WriteFile writes a copy of data to name; its parent directory must exist.
func (m *Mem) WriteFile(name string, data []byte) error {
	c := filepath.Clean(name)
	if !m.dirs[filepath.Dir(c)] {
		return notExist("open", name)
	}
	m.files[c] = append([]byte(nil), data...)
	return nil
}

// Rename moves the file at oldpath to newpath; newpath's parent must exist.
func (m *Mem) Rename(oldpath, newpath string) error {
	co := filepath.Clean(oldpath)
	data, ok := m.files[co]
	if !ok {
		return notExist("rename", oldpath)
	}
	cn := filepath.Clean(newpath)
	if !m.dirs[filepath.Dir(cn)] {
		return notExist("rename", newpath)
	}
	delete(m.files, co)
	m.files[cn] = data
	return nil
}

// Remove deletes the file or empty directory at name.
func (m *Mem) Remove(name string) error {
	c := filepath.Clean(name)
	if _, ok := m.files[c]; ok {
		delete(m.files, c)
		return nil
	}
	if m.dirs[c] {
		delete(m.dirs, c)
		return nil
	}
	return notExist("remove", name)
}

// MkdirAll marks path and every ancestor as an existing directory.
func (m *Mem) MkdirAll(path string) error {
	for d := filepath.Clean(path); ; {
		m.dirs[d] = true
		parent := filepath.Dir(d)
		if parent == d {
			return nil
		}
		d = parent
	}
}

// ReadDir lists the immediate children of name, sorted by filename.
func (m *Mem) ReadDir(name string) ([]fs.DirEntry, error) {
	c := filepath.Clean(name)
	if !m.dirs[c] {
		return nil, notExist("open", name)
	}
	var entries []fs.DirEntry
	for f := range m.files {
		if filepath.Dir(f) == c {
			entries = append(entries, fs.FileInfoToDirEntry(m.info(f)))
		}
	}
	for d := range m.dirs {
		if d != c && filepath.Dir(d) == c {
			entries = append(entries, fs.FileInfoToDirEntry(m.info(d)))
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// Stat returns the FileInfo describing name.
func (m *Mem) Stat(name string) (fs.FileInfo, error) {
	c := filepath.Clean(name)
	if _, ok := m.files[c]; !ok && !m.dirs[c] {
		return nil, notExist("stat", name)
	}
	return m.info(c), nil
}

// WalkDir walks the tree rooted at root in lexical order, honouring fs.SkipDir
// and fs.SkipAll exactly as filepath.WalkDir does.
func (m *Mem) WalkDir(root string, fn fs.WalkDirFunc) error {
	c := filepath.Clean(root)
	var err error
	if _, ok := m.files[c]; !ok && !m.dirs[c] {
		err = fn(root, nil, notExist("lstat", root))
	} else {
		err = m.walkDir(root, fs.FileInfoToDirEntry(m.info(c)), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

// walkDir recurses one node, mirroring the standard library's control flow.
func (m *Mem) walkDir(path string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(path, d, nil); err != nil || !d.IsDir() {
		if err == fs.SkipDir && d.IsDir() {
			return nil
		}
		return err
	}
	entries, err := m.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := m.walkDir(filepath.Join(path, e.Name()), e, fn); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// info returns the FileInfo for an existing clean path.
func (m *Mem) info(c string) memInfo {
	if data, ok := m.files[c]; ok {
		return memInfo{name: filepath.Base(c), size: int64(len(data))}
	}
	return memInfo{name: filepath.Base(c), dir: true}
}

// notExist builds a PathError that satisfies errors.Is(err, fs.ErrNotExist).
func notExist(op, name string) error {
	return &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
}

// memInfo is the fs.FileInfo of an in-memory file or directory.
type memInfo struct {
	name string
	size int64
	dir  bool
}

// Name returns the base name of the file.
func (i memInfo) Name() string { return i.name }

// Size returns the length in bytes, zero for a directory.
func (i memInfo) Size() int64 { return i.size }

// Mode returns 0644 for a file, ModeDir|0755 for a directory.
func (i memInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

// ModTime returns the zero time; the Mem adapter tracks no timestamps.
func (i memInfo) ModTime() time.Time { return time.Time{} }

// IsDir reports whether the entry is a directory.
func (i memInfo) IsDir() bool { return i.dir }

// Sys returns nil.
func (i memInfo) Sys() any { return nil }
