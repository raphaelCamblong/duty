// Package fsutil provides atomic filesystem writes. Every file write in duty
// goes through it so concurrent readers never see a half-written file.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path atomically: it writes a temporary file in
// the same directory as path, then renames it over the target. The resulting
// file has 0644 permissions.
func WriteAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".duty-*")
	if err != nil {
		return fmt.Errorf("write atomic %s: %w", path, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write atomic %s: %w", path, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("write atomic %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write atomic %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write atomic %s: %w", path, err)
	}
	return nil
}
