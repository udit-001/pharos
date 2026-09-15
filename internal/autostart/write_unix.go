//go:build !windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// writeEntry writes the macOS plist or Linux desktop file. Both are plain
// text files written atomically-ish (temp+rename via os.WriteFile would be
// overkill for a login entry; direct write is what the shell tools do).
func (m *Manager) writeEntry(daemonArgs []string) error {
	var content []byte
	switch runtime.GOOS {
	case "darwin":
		content = plistContent(m.exe, daemonArgs)
	case "linux":
		content = desktopContent(m.exe, daemonArgs)
	default:
		return fmt.Errorf("autostart is not supported on %s", runtime.GOOS)
	}
	if err := os.MkdirAll(filepath.Dir(m.entryPath), 0o755); err != nil {
		return fmt.Errorf("create autostart directory: %w", err)
	}
	if err := os.WriteFile(m.entryPath, content, 0o644); err != nil {
		return fmt.Errorf("write autostart entry: %w", err)
	}
	return nil
}

// readEntry reports whether the entry exists and the port pinned in its
// command line (0 when the entry exists but the port can't be determined).
func (m *Manager) readEntry() (exists bool, port int, err error) {
	data, err := os.ReadFile(m.entryPath)
	if os.IsNotExist(err) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	var a []string
	switch runtime.GOOS {
	case "darwin":
		a = parsePlistArgs(string(data))
	case "linux":
		a = parseDesktopArgs(string(data))
	default:
		return false, 0, fmt.Errorf("autostart is not supported on %s", runtime.GOOS)
	}
	return true, portFromArgs(a), nil
}
