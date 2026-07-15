package tests

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/raphaelCamblong/duty/internal/fsys"
)

// fsFactory names an FS adapter and builds a fresh instance whose root
// directory already exists.
type fsFactory struct {
	name string
	make func(t *testing.T) (fsys.FS, string)
}

// fsAdapters is every FS adapter under test; both must pass one shared table.
var fsAdapters = []fsFactory{
	{
		name: "OS",
		make: func(t *testing.T) (fsys.FS, string) { return fsys.OS{}, t.TempDir() },
	},
	{
		name: "Mem",
		make: func(t *testing.T) (fsys.FS, string) {
			m := fsys.NewMem()
			root := "/work"
			if err := m.MkdirAll(root); err != nil {
				t.Fatalf("MkdirAll(%q): %v", root, err)
			}
			return m, root
		},
	},
}

// fsBehavior is one shared assertion the OS and Mem adapters must both satisfy.
type fsBehavior struct {
	name string
	run  func(t *testing.T, f fsys.FS, root string)
}

var fsBehaviors = []fsBehavior{
	{
		name: "write then read round-trips as mode 0644",
		run: func(t *testing.T, f fsys.FS, root string) {
			p := filepath.Join(root, "a.md")
			if err := f.WriteFile(p, []byte("hello\n")); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			got, err := f.ReadFile(p)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if string(got) != "hello\n" {
				t.Errorf("read %q, want %q", got, "hello\n")
			}
			info, err := f.Stat(p)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if info.IsDir() {
				t.Error("Stat reports a directory, want a file")
			}
			if perm := info.Mode().Perm(); perm != 0o644 {
				t.Errorf("perm = %04o, want 0644", perm)
			}
		},
	},
	{
		name: "atomic replace overwrites and leaves no residue",
		run: func(t *testing.T, f fsys.FS, root string) {
			p := filepath.Join(root, "a.md")
			if err := f.WriteFile(p, []byte("old content, deliberately long\n")); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if err := f.WriteFile(p, []byte("new\n")); err != nil {
				t.Fatalf("replace: %v", err)
			}
			got, err := f.ReadFile(p)
			if err != nil || string(got) != "new\n" {
				t.Fatalf("read after replace = %q, %v, want %q", got, err, "new\n")
			}
			entries, err := f.ReadDir(root)
			if err != nil {
				t.Fatalf("ReadDir: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != "a.md" {
				var names []string
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Errorf("dir entries = %v, want [a.md] only (temp residue?)", names)
			}
		},
	},
	{
		name: "rename moves the file and clears the source",
		run: func(t *testing.T, f fsys.FS, root string) {
			src := filepath.Join(root, "src.md")
			dst := filepath.Join(root, "dst.md")
			if err := f.WriteFile(src, []byte("payload\n")); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if err := f.Rename(src, dst); err != nil {
				t.Fatalf("Rename: %v", err)
			}
			if _, err := f.ReadFile(src); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("source after rename: err = %v, want ErrNotExist", err)
			}
			got, err := f.ReadFile(dst)
			if err != nil || string(got) != "payload\n" {
				t.Errorf("dst after rename = %q, %v, want %q", got, err, "payload\n")
			}
		},
	},
	{
		name: "walk visits every node in lexical order",
		run: func(t *testing.T, f fsys.FS, root string) {
			if err := f.MkdirAll(filepath.Join(root, "b")); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
			for _, p := range []string{"a.md", "z.md", filepath.Join("b", "c.md")} {
				if err := f.WriteFile(filepath.Join(root, p), []byte("x")); err != nil {
					t.Fatalf("seed %s: %v", p, err)
				}
			}
			var got []string
			err := f.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				got = append(got, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				t.Fatalf("WalkDir: %v", err)
			}
			want := []string{".", "a.md", "b", "b/c.md", "z.md"}
			if len(got) != len(want) {
				t.Fatalf("walk = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("walk[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		},
	},
	{
		name: "reading a missing file is fs.ErrNotExist",
		run: func(t *testing.T, f fsys.FS, root string) {
			if _, err := f.ReadFile(filepath.Join(root, "nope.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("ReadFile(missing): err = %v, want ErrNotExist", err)
			}
		},
	},
	{
		name: "writing into a missing directory errors",
		run: func(t *testing.T, f fsys.FS, root string) {
			if err := f.WriteFile(filepath.Join(root, "no-such-dir", "a.md"), []byte("x")); err == nil {
				t.Error("WriteFile into a missing directory: want error, got nil")
			}
		},
	},
}

func TestFSAdapters(t *testing.T) {
	for _, a := range fsAdapters {
		t.Run(a.name, func(t *testing.T) {
			for _, b := range fsBehaviors {
				t.Run(b.name, func(t *testing.T) {
					f, root := a.make(t)
					b.run(t, f, root)
				})
			}
		})
	}
}

func TestLock(t *testing.T) {
	t.Run("OS serializes a second acquire until the first releases", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".duty.lock")
		unlock, err := fsys.OS{}.Lock(path)
		if err != nil {
			t.Fatalf("first lock: %v", err)
		}
		acquired := make(chan struct{})
		go func() {
			u2, err := fsys.OS{}.Lock(path)
			if err != nil {
				t.Errorf("second lock: %v", err)
				return
			}
			u2()
			close(acquired)
		}()
		select {
		case <-acquired:
			t.Fatal("second lock acquired while the first was held")
		case <-time.After(100 * time.Millisecond):
		}
		unlock()
		select {
		case <-acquired:
		case <-time.After(2 * time.Second):
			t.Fatal("second lock never acquired after release")
		}
	})

	t.Run("Mem reports the tree locked when held past the timeout", func(t *testing.T) {
		m := fsys.NewMem()
		m.LockTimeout = 30 * time.Millisecond
		unlock, err := m.Lock("/.duty.lock")
		if err != nil {
			t.Fatalf("first lock: %v", err)
		}
		defer unlock()
		if _, err := m.Lock("/.duty.lock"); err == nil || err.Error() != "tree is locked" {
			t.Errorf("second lock err = %v, want \"tree is locked\"", err)
		}
	})

	t.Run("a released lock can be re-acquired", func(t *testing.T) {
		for _, a := range fsAdapters {
			t.Run(a.name, func(t *testing.T) {
				f, root := a.make(t)
				path := filepath.Join(root, ".duty.lock")
				unlock, err := f.Lock(path)
				if err != nil {
					t.Fatalf("first lock: %v", err)
				}
				unlock()
				u2, err := f.Lock(path)
				if err != nil {
					t.Fatalf("re-lock after release: %v", err)
				}
				u2()
			})
		}
	})
}
