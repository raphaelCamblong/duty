package fsys

import (
	"io/fs"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// memLockWait is the default acquire timeout for Mem.Lock; tests override it
// through LockTimeout to exercise the "tree is locked" path quickly.
const memLockWait = 5 * time.Second

// Mem is a map-backed in-memory FS for fast tests. Files carry mode 0644,
// directories 0755; a file's parent directory must exist before it is written.
type Mem struct {
	files map[string][]byte
	dirs  map[string]bool
	// LockTimeout bounds how long Lock blocks before reporting the tree
	// locked; zero uses memLockWait.
	LockTimeout time.Duration
	mu          sync.Mutex
	locks       map[string]chan struct{}
}

var _ FS = (*Mem)(nil)

func NewMem() *Mem {
	return &Mem{
		files: map[string][]byte{},
		dirs:  map[string]bool{filepath.Clean("/"): true},
		locks: map[string]chan struct{}{},
	}
}

func (m *Mem) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[filepath.Clean(name)]
	if !ok {
		return nil, notExist("open", name)
	}
	return append([]byte(nil), data...), nil
}

func (m *Mem) WriteFile(name string, data []byte) error {
	clean := filepath.Clean(name)
	if !m.dirs[filepath.Dir(clean)] {
		return notExist("open", name)
	}
	m.files[clean] = append([]byte(nil), data...)
	return nil
}

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

func (m *Mem) Remove(name string) error {
	clean := filepath.Clean(name)
	if _, ok := m.files[clean]; ok {
		delete(m.files, clean)
		return nil
	}
	if m.dirs[clean] {
		delete(m.dirs, clean)
		return nil
	}
	return notExist("remove", name)
}

func (m *Mem) MkdirAll(path string) error {
	for dir := filepath.Clean(path); ; {
		m.dirs[dir] = true
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func (m *Mem) ReadDir(name string) ([]fs.DirEntry, error) {
	clean := filepath.Clean(name)
	if !m.dirs[clean] {
		return nil, notExist("open", name)
	}
	var entries []fs.DirEntry
	for file := range m.files {
		if filepath.Dir(file) == clean {
			entries = append(entries, fs.FileInfoToDirEntry(m.info(file)))
		}
	}
	for dir := range m.dirs {
		if dir != clean && filepath.Dir(dir) == clean {
			entries = append(entries, fs.FileInfoToDirEntry(m.info(dir)))
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (m *Mem) Stat(name string) (fs.FileInfo, error) {
	clean := filepath.Clean(name)
	if !m.exists(clean) {
		return nil, notExist("stat", name)
	}
	return m.info(clean), nil
}

// WalkDir walks the tree rooted at root in lexical order, honouring fs.SkipDir
// and fs.SkipAll exactly as filepath.WalkDir does.
func (m *Mem) WalkDir(root string, fn fs.WalkDirFunc) error {
	clean := filepath.Clean(root)
	var err error
	if !m.exists(clean) {
		err = fn(root, nil, notExist("lstat", root))
	} else {
		err = m.walkDir(root, fs.FileInfoToDirEntry(m.info(clean)), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

// walkDir recurses one node, mirroring the standard library's control flow.
func (m *Mem) walkDir(path string, entry fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(path, entry, nil); err != nil || !entry.IsDir() {
		if err == fs.SkipDir && entry.IsDir() {
			return nil
		}
		return err
	}
	entries, err := m.ReadDir(path)
	if err != nil {
		return err
	}
	for _, child := range entries {
		err := m.walkDir(filepath.Join(path, child.Name()), child, fn)
		if err == fs.SkipDir {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Lock acquires the per-path in-process lock, blocking up to LockTimeout
// (memLockWait when zero) before reporting the tree held elsewhere.
func (m *Mem) Lock(path string) (func(), error) {
	ch := m.lockChan(path)
	wait := m.LockTimeout
	if wait == 0 {
		wait = memLockWait
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-timer.C:
		return nil, errLocked
	}
}

func (m *Mem) lockChan(path string) chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := filepath.Clean(path)
	ch, ok := m.locks[clean]
	if !ok {
		ch = make(chan struct{}, 1)
		m.locks[clean] = ch
	}
	return ch
}

func (m *Mem) exists(clean string) bool {
	_, ok := m.files[clean]
	return ok || m.dirs[clean]
}

// info returns the FileInfo for an existing clean path.
func (m *Mem) info(clean string) memInfo {
	if data, ok := m.files[clean]; ok {
		return memInfo{name: filepath.Base(clean), size: int64(len(data))}
	}
	return memInfo{name: filepath.Base(clean), dir: true}
}

// notExist builds a PathError that satisfies errors.Is(err, fs.ErrNotExist).
func notExist(op, name string) error {
	return &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
}

type memInfo struct {
	name string
	size int64
	dir  bool
}

func (i memInfo) Name() string { return i.name }

// Size returns the length in bytes, zero for a directory.
func (i memInfo) Size() int64 { return i.size }

func (i memInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

// ModTime returns the zero time; the Mem adapter tracks no timestamps.
func (i memInfo) ModTime() time.Time { return time.Time{} }

func (i memInfo) IsDir() bool { return i.dir }

func (i memInfo) Sys() any { return nil }
