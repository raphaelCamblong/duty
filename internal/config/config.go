// Package config loads duty's TOML configuration, merging the project file
// over the user file over built-in defaults. Zero configuration always
// works: missing files are skipped and every key has a default.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/raphaelCamblong/duty/internal/fsys"
)

// Config holds duty's tunable settings. Config tunes presentation only;
// statuses, naming, and board structure are the convention, never settings.
type Config struct {
	// Editor is the command used to open task files.
	Editor string `toml:"editor"`
	// TUI groups terminal-UI settings.
	TUI struct {
		// Theme selects the TUI color scheme: auto, dark, or light.
		Theme string `toml:"theme"`
		// Palette overrides individual color slots of the TUI theme.
		Palette Palette `toml:"palette"`
	} `toml:"tui"`
}

// Palette holds optional TUI color overrides, one per semantic slot; a nil
// slot keeps the built-in default. The color model itself lives in the TUI —
// config keeps the values as plain strings so this layer stays presentation-free.
type Palette struct {
	// Accent overrides the accent slot (focused chrome, ids, breadcrumb).
	Accent *Color `toml:"accent"`
	// Dim overrides the dim slot (separators, ages, hints).
	Dim *Color `toml:"dim"`
	// Todo overrides the todo status slot.
	Todo *Color `toml:"todo"`
	// InProgress overrides the in-progress status slot.
	InProgress *Color `toml:"in-progress"`
	// Done overrides the done status slot.
	Done *Color `toml:"done"`
	// Blocked overrides the blocked status slot (also scan errors and drift).
	Blocked *Color `toml:"blocked"`
}

// Color is one themeable color read from config: a bare string sets both the
// light and dark channels, or an inline table { light = "…", dark = "…" } sets
// them independently. The values stay unvalidated here; the TUI checks them.
type Color struct {
	// Light is the value used on a light background.
	Light string
	// Dark is the value used on a dark background.
	Dark string
}

// UnmarshalTOML reads a Color from a bare string (both channels) or an inline
// table with light and dark keys.
func (c *Color) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case string:
		c.Light, c.Dark = val, val
		return nil
	case map[string]any:
		c.Light, _ = val["light"].(string)
		c.Dark, _ = val["dark"].(string)
		return nil
	default:
		return fmt.Errorf("theme color must be a string or { light, dark } table, got %T", v)
	}
}

// Load reads the TOML files at userPath and projectPath and merges them over
// the built-in defaults, key by key: project overrides user overrides
// defaults. An empty path or a missing file is skipped; a malformed file is
// an error.
func Load(f fsys.FS, userPath, projectPath string) (Config, error) {
	cfg := defaults()
	if err := mergeFile(f, &cfg, userPath); err != nil {
		return Config{}, err
	}
	if err := mergeFile(f, &cfg, projectPath); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// UserPath returns the user-level config file location:
// os.UserConfigDir()/duty/config.toml.
func UserPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, "duty", "config.toml"), nil
}

// defaults returns the built-in configuration: Editor from $EDITOR (vi when
// unset or empty) and Theme "auto".
func defaults() Config {
	cfg := Config{Editor: os.Getenv("EDITOR")}
	if cfg.Editor == "" {
		cfg.Editor = "vi"
	}
	cfg.TUI.Theme = "auto"
	return cfg
}

// mergeFile decodes the TOML file at path into cfg, overriding only the keys
// the file sets. An empty path or a missing file leaves cfg untouched.
func mergeFile(f fsys.FS, cfg *Config, path string) error {
	if path == "" {
		return nil
	}
	data, err := f.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}
