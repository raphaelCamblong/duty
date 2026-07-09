package tests

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// buildTree materializes a fixture tree under a fresh t.TempDir(). Entries
// ending in "/" become directories; everything else becomes an empty file
// (the tree package reads names only). It returns the temp root.
func buildTree(t *testing.T, entries ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, entry := range entries {
		abs := filepath.Join(root, filepath.FromSlash(entry))
		if strings.HasSuffix(entry, "/") {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", entry, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir parent of %s: %v", entry, err)
		}
		if err := os.WriteFile(abs, nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", entry, err)
		}
	}
	return root
}

func TestFindRoot(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		cwd     string // slash-relative to the temp root
		want    string // slash-relative expected root; "" with wantErr set
		wantErr bool
	}{
		{
			name: "walks up from a deep cwd to the topmost BOARD.md",
			entries: []string{
				"duty/BOARD.md",
				"duty/backend/BOARD.md",
				"duty/backend/api/BOARD.md",
			},
			cwd:  "duty/backend/api",
			want: "duty",
		},
		{
			name: "cwd in a plain subdirectory below a board",
			entries: []string{
				"duty/BOARD.md",
				"duty/backend/BOARD.md",
				"duty/backend/scratch/",
			},
			cwd:  "duty/backend/scratch",
			want: "duty",
		},
		{
			name: "duty.toml marks the root and stops the ascent",
			entries: []string{
				"outer/BOARD.md",
				"outer/inner/BOARD.md",
				"outer/inner/duty.toml",
				"outer/inner/deep/BOARD.md",
			},
			cwd:  "outer/inner/deep",
			want: "outer/inner",
		},
		{
			name:    "cwd is itself the root board",
			entries: []string{"duty/BOARD.md"},
			cwd:     "duty",
			want:    "duty",
		},
		{
			name: "ascent stops at a directory without BOARD.md",
			entries: []string{
				"a/BOARD.md",
				"a/b/",
				"a/b/c/BOARD.md",
			},
			cwd:  "a/b/c",
			want: "a/b/c",
		},
		{
			name:    "outside a tree falls back to ./duty",
			entries: []string{"duty/BOARD.md"},
			cwd:     ".",
			want:    "duty",
		},
		{
			name:    "outside a tree with no ./duty errors",
			entries: []string{"src/"},
			cwd:     "src",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := buildTree(t, tt.entries...)
			got, err := tree.FindRoot(fsys.OS{}, filepath.Join(root, filepath.FromSlash(tt.cwd)))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("FindRoot() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindRoot() error = %v", err)
			}
			want := filepath.Join(root, filepath.FromSlash(tt.want))
			if got != want {
				t.Errorf("FindRoot() = %q, want %q", got, want)
			}
		})
	}
}

func TestCurrentBoard(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		cwd     string
		want    string
		wantErr bool
	}{
		{
			name: "nearest ancestor board wins over the root",
			entries: []string{
				"duty/BOARD.md",
				"duty/backend/BOARD.md",
			},
			cwd:  "duty/backend",
			want: "duty/backend",
		},
		{
			name: "plain subdirectory resolves to its enclosing board",
			entries: []string{
				"duty/BOARD.md",
				"duty/backend/BOARD.md",
				"duty/backend/scratch/",
			},
			cwd:  "duty/backend/scratch",
			want: "duty/backend",
		},
		{
			name:    "outside a tree falls back to ./duty",
			entries: []string{"duty/BOARD.md"},
			cwd:     ".",
			want:    "duty",
		},
		{
			name:    "outside a tree with no ./duty errors",
			entries: []string{"src/"},
			cwd:     "src",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := buildTree(t, tt.entries...)
			got, err := tree.CurrentBoard(fsys.OS{}, filepath.Join(root, filepath.FromSlash(tt.cwd)))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("CurrentBoard() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("CurrentBoard() error = %v", err)
			}
			want := filepath.Join(root, filepath.FromSlash(tt.want))
			if got != want {
				t.Errorf("CurrentBoard() = %q, want %q", got, want)
			}
		})
	}
}

func TestBoards(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		want    []string // slash-relative to the tree root at duty/
		wantErr string   // substring of the expected error; "" means success
	}{
		{
			name:    "root only",
			entries: []string{"duty/BOARD.md"},
			want:    []string{"."},
		},
		{
			name: "collects nested boards, skips archive and plain dirs",
			entries: []string{
				"duty/BOARD.md",
				"duty/archive/BOARD.md",
				"duty/backend/BOARD.md",
				"duty/backend/archive/T-02-old.md",
				"duty/backend/api/BOARD.md",
				"duty/notes/scratch.md",
			},
			want: []string{".", "backend", "backend/api"},
		},
		{
			name: "duty.toml at the root is fine",
			entries: []string{
				"duty/BOARD.md",
				"duty/duty.toml",
				"duty/backend/BOARD.md",
			},
			want: []string{".", "backend"},
		},
		{
			name: "duty.toml below the root is a second-root error",
			entries: []string{
				"duty/BOARD.md",
				"duty/backend/BOARD.md",
				"duty/backend/duty.toml",
			},
			wantErr: "second duty.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := buildTree(t, tt.entries...)
			root := filepath.Join(tmp, "duty")
			got, err := tree.Boards(fsys.OS{}, root)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Boards() = %v, want error containing %q", got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Boards() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Boards() error = %v", err)
			}
			want := make([]string, 0, len(tt.want))
			for _, rel := range tt.want {
				want = append(want, filepath.Join(root, filepath.FromSlash(rel)))
			}
			if len(got) != len(want) {
				t.Fatalf("Boards() = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("Boards()[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}

func TestResolveTask(t *testing.T) {
	entries := []string{
		"duty/BOARD.md",
		"duty/T-01-bootstrap.md",
		"duty/archive/T-03-retired.md",
		"duty/backend/BOARD.md",
		"duty/backend/T-02-endpoints.md",
		"duty/backend/T-10-migrations.md",
	}

	tests := []struct {
		name         string
		id           string
		want         string // slash-relative to the tree root; "" for errors
		wantArchived bool
		wantErr      bool
	}{
		{
			name: "open task at the root",
			id:   "T-01",
			want: "duty/T-01-bootstrap.md",
		},
		{
			name: "open task in a sub-board",
			id:   "T-02",
			want: "duty/backend/T-02-endpoints.md",
		},
		{
			name:         "archived task is a read-only error naming the id",
			id:           "T-03",
			wantArchived: true,
			wantErr:      true,
		},
		{
			name:    "unknown id",
			id:      "T-99",
			wantErr: true,
		},
		{
			name:    "id is not a prefix match on longer numbers",
			id:      "T-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := buildTree(t, entries...)
			root := filepath.Join(tmp, "duty")
			got, err := tree.ResolveTask(fsys.OS{}, root, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveTask(%q) = %q, want error", tt.id, got)
				}
				if archived := errors.Is(err, tree.ErrArchived); archived != tt.wantArchived {
					t.Fatalf("ResolveTask(%q) errors.Is(ErrArchived) = %v, want %v (err: %v)", tt.id, archived, tt.wantArchived, err)
				}
				if !strings.Contains(err.Error(), tt.id) {
					t.Errorf("ResolveTask(%q) error = %v, want it to name the id", tt.id, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveTask(%q) error = %v", tt.id, err)
			}
			want := filepath.Join(tmp, filepath.FromSlash(tt.want))
			if got != want {
				t.Errorf("ResolveTask(%q) = %q, want %q", tt.id, got, want)
			}
		})
	}
}

func TestIsTaskFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "T-01-bootstrap.md", want: true},
		{name: "T-120-big.md", want: true},
		{name: "BOARD.md", want: false},
		{name: "README.md", want: false},
		{name: "T-01-bootstrap.txt", want: false},
		{name: "T-bootstrap.md", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tree.IsTaskFile(tt.name); got != tt.want {
				t.Errorf("IsTaskFile(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestNextNN(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		want    string
	}{
		{
			name:    "empty tree starts at 01",
			entries: []string{"duty/BOARD.md"},
			want:    "01",
		},
		{
			name: "next after the highest open task",
			entries: []string{
				"duty/BOARD.md",
				"duty/T-01-first.md",
				"duty/T-03-third.md",
			},
			want: "04",
		},
		{
			name: "archived numbers block reuse",
			entries: []string{
				"duty/BOARD.md",
				"duty/T-01-first.md",
				"duty/archive/T-07-retired.md",
			},
			want: "08",
		},
		{
			name: "counts every board including sub-board archives",
			entries: []string{
				"duty/BOARD.md",
				"duty/T-02-second.md",
				"duty/backend/BOARD.md",
				"duty/backend/T-05-fifth.md",
				"duty/backend/archive/T-09-retired.md",
			},
			want: "10",
		},
		{
			name: "grows naturally past two digits",
			entries: []string{
				"duty/BOARD.md",
				"duty/T-120-big.md",
			},
			want: "121",
		},
		{
			name: "ignores files that are not task files",
			entries: []string{
				"duty/BOARD.md",
				"duty/README.md",
				"duty/T-02-second.md",
				"duty/notes/T-99-fake.txt",
			},
			want: "03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := buildTree(t, tt.entries...)
			got, err := tree.NextNN(fsys.OS{}, filepath.Join(tmp, "duty"))
			if err != nil {
				t.Fatalf("NextNN() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("NextNN() = %q, want %q", got, tt.want)
			}
		})
	}
}
