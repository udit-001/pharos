//go:build windows

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The .lnk startup launch of a console-subsystem exe flashes a console
// window for ~0.5s at each login before the daemon detaches and the window
// disappears. This is ACCEPTED by design (LEARN-165 #280, user decision):
// no Task Scheduler hidden launch, no windowsgui stub, no --app. The flash
// is the price of the single daemon code path. Revisit only if the
// enterprise user complains — and reopen the decision before "fixing" it.

// writeEntry creates/rewrites the Startup-folder shortcut through the same
// PowerShell WScript.Shell seam we read with. ShellExecute launches the .lnk
// detached, so pidfile + stdout/stderr redirect behavior matches the console
// path.
func (m *Manager) writeEntry(daemonArgs []string) error {
	if err := os.MkdirAll(filepath.Dir(m.entryPath), 0o755); err != nil {
		return fmt.Errorf("create Startup folder shortcut dir: %w", err)
	}
	script := lnkScript(m.exe, m.entryPath, portFromArgs(daemonArgs))
	if err := exec.Command("powershell", "-NoProfile", "-Command", script).Run(); err != nil {
		return fmt.Errorf("create startup shortcut: %w", err)
	}
	return nil
}

// readEntry reports whether the .lnk exists and the port pinned in its
// Arguments (read back through the same PowerShell seam — the .lnk is a
// binary format). When PowerShell is blocked (AppLocker/AV), the entry is
// reported as existing with an unknown port: status shows "enabled", never
// falsely "disabled" — the spec's text-level policy-blocked caveat.
func (m *Manager) readEntry() (exists bool, port int, err error) {
	if _, err := os.Stat(m.entryPath); os.IsNotExist(err) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}
	script := "(New-Object -ComObject WScript.Shell).CreateShortcut(" + psQuote(m.entryPath) + ").Arguments"
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return true, 0, nil
	}
	return true, portFromArgs(strings.Fields(strings.TrimSpace(string(out)))), nil
}
