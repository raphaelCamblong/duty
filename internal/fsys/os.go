package fsys

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// lockWait bounds how long Lock blocks for the tree lock before reporting it
// held elsewhere; lockRetryDelay is how often it re-tries while waiting.
const (
	lockWait       = 5 * time.Second
	lockRetryDelay = 20 * time.Millisecond
)

type OS struct{}

var _ FS = OS{}

func (OS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

func (OS) Rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }

func (OS) Remove(name string) error { return os.Remove(name) }

func (OS) MkdirAll(path string) error { return os.MkdirAll(path, 0o755) }

func (OS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }

func (OS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

func (OS) WalkDir(root string, fn fs.WalkDirFunc) error { return filepath.WalkDir(root, fn) }

// Lock takes an exclusive flock on path, blocking up to lockWait, and returns
// a release function. A wait that times out reports the tree held elsewhere.
func (OS) Lock(path string) (func(), error) {
	fl := flock.New(path)
	ctx, cancel := context.WithTimeout(context.Background(), lockWait)
	defer cancel()
	locked, err := fl.TryLockContext(ctx, lockRetryDelay)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errLocked
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	if !locked {
		return nil, errLocked
	}
	return func() { _ = fl.Unlock() }, nil
}

// WriteFile writes data to name atomically: a temp file in the same directory,
// then a rename over the target. The resulting file has 0644 permissions.
func (OS) WriteFile(name string, data []byte) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("write %s: %w", name, err)
		}
	}()
	tmp, err := os.CreateTemp(filepath.Dir(name), ".duty-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}
