package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raphaelCamblong/duty/internal/fsutil"
)

func TestWriteAtomic(t *testing.T) {
	tests := []struct {
		name     string
		existing string // pre-existing target content; "" means no file
		data     string
	}{
		{
			name: "creates a new file",
			data: "hello\n",
		},
		{
			name:     "fully replaces an existing file",
			existing: "old content, deliberately longer than the replacement\n",
			data:     "new\n",
		},
		{
			name:     "replaces with empty content",
			existing: "something\n",
			data:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "target.md")
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), 0o600); err != nil {
					t.Fatalf("seed existing file: %v", err)
				}
			}

			if err := fsutil.WriteAtomic(path, []byte(tt.data)); err != nil {
				t.Fatalf("WriteAtomic() error = %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back target: %v", err)
			}
			if string(got) != tt.data {
				t.Errorf("content = %q, want %q", got, tt.data)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat target: %v", err)
			}
			if perm := info.Mode().Perm(); perm != 0o644 {
				t.Errorf("perm = %04o, want 0644", perm)
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("read dir: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != "target.md" {
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Errorf("directory entries = %v, want [target.md] only (temp-file residue?)", names)
			}
		})
	}
}

func TestWriteAtomicMissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "target.md")
	if err := fsutil.WriteAtomic(path, []byte("data")); err == nil {
		t.Fatal("WriteAtomic() into a missing directory: want error, got nil")
	}
}
