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
	} `toml:"tui"`
}

// Load reads the TOML files at userPath and projectPath and merges them over
// the built-in defaults, key by key: project overrides user overrides
// defaults. An empty path or a missing file is skipped; a malformed file is
// an error.
func Load(userPath, projectPath string) (Config, error) {
	cfg := defaults()
	if err := mergeFile(&cfg, userPath); err != nil {
		return Config{}, err
	}
	if err := mergeFile(&cfg, projectPath); err != nil {
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
func mergeFile(cfg *Config, path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
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
