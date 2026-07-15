package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raphaelCamblong/duty/internal/config"
	"github.com/raphaelCamblong/duty/internal/fsys"
)

func TestLoad(t *testing.T) {
	toml := func(s string) *string { return &s }

	tests := []struct {
		name       string
		editorEnv  string  // value for $EDITOR; "" means unset
		user       *string // user config content; nil means no file
		project    *string // project config content; nil means no file
		wantEditor string
		wantTheme  string
		wantErr    bool
	}{
		{
			name:       "defaults only, EDITOR set",
			editorEnv:  "nano",
			wantEditor: "nano",
			wantTheme:  "auto",
		},
		{
			name:       "defaults only, EDITOR unset falls back to vi",
			wantEditor: "vi",
			wantTheme:  "auto",
		},
		{
			name:       "user only overrides defaults",
			editorEnv:  "nano",
			user:       toml("editor = \"nvim\"\n\n[tui]\ntheme = \"dark\"\n"),
			wantEditor: "nvim",
			wantTheme:  "dark",
		},
		{
			name:       "project only overrides defaults",
			editorEnv:  "nano",
			project:    toml("editor = \"hx\"\n\n[tui]\ntheme = \"light\"\n"),
			wantEditor: "hx",
			wantTheme:  "light",
		},
		{
			name:       "project overrides user",
			editorEnv:  "nano",
			user:       toml("editor = \"nvim\"\n\n[tui]\ntheme = \"dark\"\n"),
			project:    toml("editor = \"hx\"\n\n[tui]\ntheme = \"light\"\n"),
			wantEditor: "hx",
			wantTheme:  "light",
		},
		{
			name:       "partial files merge per-key",
			user:       toml("editor = \"nvim\"\n"),
			project:    toml("[tui]\ntheme = \"light\"\n"),
			wantEditor: "nvim",
			wantTheme:  "light",
		},
		{
			name:       "partial project keeps user keys it does not set",
			editorEnv:  "nano",
			user:       toml("editor = \"nvim\"\n\n[tui]\ntheme = \"dark\"\n"),
			project:    toml("[tui]\ntheme = \"light\"\n"),
			wantEditor: "nvim",
			wantTheme:  "light",
		},
		{
			name:       "partial user keeps defaults for unset keys",
			editorEnv:  "nano",
			user:       toml("[tui]\ntheme = \"dark\"\n"),
			wantEditor: "nano",
			wantTheme:  "dark",
		},
		{
			name:       "empty files keep defaults",
			user:       toml(""),
			project:    toml(""),
			wantEditor: "vi",
			wantTheme:  "auto",
		},
		{
			name:    "malformed user TOML is an error",
			user:    toml("editor = \n"),
			wantErr: true,
		},
		{
			name:    "malformed project TOML is an error",
			user:    toml("editor = \"nvim\"\n"),
			project: toml("[tui\ntheme = \"dark\"\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editorEnv)

			dir := t.TempDir()
			userPath := filepath.Join(dir, "config.toml")
			projectPath := filepath.Join(dir, "duty.toml")
			if tt.user != nil {
				if err := os.WriteFile(userPath, []byte(*tt.user), 0o600); err != nil {
					t.Fatalf("seed user config: %v", err)
				}
			}
			if tt.project != nil {
				if err := os.WriteFile(projectPath, []byte(*tt.project), 0o600); err != nil {
					t.Fatalf("seed project config: %v", err)
				}
			}

			got, err := config.Load(fsys.OS{}, userPath, projectPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Load() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Editor != tt.wantEditor {
				t.Errorf("Editor = %q, want %q", got.Editor, tt.wantEditor)
			}
			if got.TUI.Theme != tt.wantTheme {
				t.Errorf("TUI.Theme = %q, want %q", got.TUI.Theme, tt.wantTheme)
			}
		})
	}
}

func TestLoadPalette(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "duty.toml")
	content := "[tui]\n" +
		"theme = \"dark\"\n\n" +
		"[tui.palette]\n" +
		"accent = \"#ff8800\"\n" +
		"done = { light = \"#111111\", dark = \"#eeeeee\" }\n"
	if err := os.WriteFile(projectPath, []byte(content), 0o600); err != nil {
		t.Fatalf("seed project config: %v", err)
	}

	got, err := config.Load(fsys.OS{}, "", projectPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.TUI.Theme != "dark" {
		t.Errorf("theme selector clobbered by the palette table: %q", got.TUI.Theme)
	}
	p := got.TUI.Palette
	if p.Accent == nil || p.Accent.Light != "#ff8800" || p.Accent.Dark != "#ff8800" {
		t.Errorf("bare-string accent = %+v, want both channels #ff8800", p.Accent)
	}
	if p.Done == nil || p.Done.Light != "#111111" || p.Done.Dark != "#eeeeee" {
		t.Errorf("table done = %+v, want light #111111 dark #eeeeee", p.Done)
	}
	if p.Dim != nil || p.Todo != nil || p.InProgress != nil || p.Blocked != nil {
		t.Errorf("unset slots should stay nil, got dim=%v todo=%v inprog=%v blocked=%v",
			p.Dim, p.Todo, p.InProgress, p.Blocked)
	}
}

func TestLoadEmptyPaths(t *testing.T) {
	t.Setenv("EDITOR", "nano")

	got, err := config.Load(fsys.OS{}, "", "")
	if err != nil {
		t.Fatalf("Load(\"\", \"\") error = %v", err)
	}
	if got.Editor != "nano" {
		t.Errorf("Editor = %q, want %q", got.Editor, "nano")
	}
	if got.TUI.Theme != "auto" {
		t.Errorf("TUI.Theme = %q, want %q", got.TUI.Theme, "auto")
	}
}

func TestUserPath(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir() unavailable: %v", err)
	}

	got, err := config.UserPath()
	if err != nil {
		t.Fatalf("UserPath() error = %v", err)
	}
	want := filepath.Join(base, "duty", "config.toml")
	if got != want {
		t.Errorf("UserPath() = %q, want %q", got, want)
	}
}
