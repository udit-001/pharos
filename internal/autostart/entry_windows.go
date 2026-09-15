//go:build windows

package autostart

import (
	"errors"
	"os"
	"path/filepath"
)

// entryPath is the per-user Startup folder shortcut — logged-in user only,
// no elevation (LEARN-165 #261).
func entryPath() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", errors.New("APPDATA is not set — cannot locate the Startup folder")
	}
	return filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "pharos.lnk"), nil
}
