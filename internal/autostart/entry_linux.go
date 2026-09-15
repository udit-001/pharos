//go:build linux

package autostart

import (
	"os"
	"path/filepath"
)

// entryPath is the XDG autostart desktop file. os.UserConfigDir honors
// $XDG_CONFIG_HOME (falling back to ~/.config), which keeps tests real: a
// temp XDG_CONFIG_HOME puts the entry in a temp dir, and the install path
// matches the spec's ~/.config/autostart/pharos.desktop.
func entryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "pharos.desktop"), nil
}
