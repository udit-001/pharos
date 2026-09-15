//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
)

// entryPath is the per-user launchd agent — no root daemon (LEARN-165 #261).
func entryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.udit001.pharos.plist"), nil
}
