// Package config loads duty's TOML configuration: project over user over built-in defaults.
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

// Config tunes presentation only — statuses, naming, and board structure are the convention, never settings.
type Config struct {
	Editor string `toml:"editor"`
	TUI    struct {
		// Theme selects the TUI color scheme: auto, dark, or light.
		Theme   string  `toml:"theme"`
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
	Dim        *Color `toml:"dim"`
	Todo       *Color `toml:"todo"`
	InProgress *Color `toml:"in-progress"`
	Done       *Color `toml:"done"`
	// Blocked overrides the blocked status slot (also scan errors and drift).
	Blocked *Color `toml:"blocked"`
}

// Color is one themeable color read from config: a bare string sets both the
// light and dark channels, or an inline table { light = "…", dark = "…" } sets
// them independently. The values stay unvalidated here; the TUI checks them.
type Color struct {
	Light string
	Dark  string
}

// UnmarshalTOML reads a Color from a bare string (both channels) or an inline
// table with light and dark keys.
func (c *Color) UnmarshalTOML(value any) error {
	switch val := value.(type) {
	case string:
		c.Light, c.Dark = val, val
		return nil
	case map[string]any:
		c.Light, _ = val["light"].(string)
		c.Dark, _ = val["dark"].(string)
		return nil
	default:
		return fmt.Errorf("theme color must be a string or { light, dark } table, got %T", value)
	}
}

// Load reads the TOML files at userPath and projectPath and merges them over
// the built-in defaults, key by key: project overrides user overrides
// defaults. An empty path or a missing file is skipped; a malformed file is
// an error.
func Load(filesystem fsys.FS, userPath, projectPath string) (Config, error) {
	cfg := defaults()
	if err := mergeFile(filesystem, &cfg, userPath); err != nil {
		return Config{}, err
	}
	if err := mergeFile(filesystem, &cfg, projectPath); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func UserPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, "duty", "config.toml"), nil
}

func defaults() Config {
	cfg := Config{Editor: os.Getenv("EDITOR")}
	if cfg.Editor == "" {
		cfg.Editor = "vi"
	}
	cfg.TUI.Theme = "auto"
	return cfg
}

// mergeFile decodes the TOML file at path into cfg, overwriting only the keys the file sets.
func mergeFile(filesystem fsys.FS, cfg *Config, path string) error {
	if path == "" {
		return nil
	}
	data, err := filesystem.ReadFile(path)
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
