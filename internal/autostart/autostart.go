// Package autostart manages the per-user startup entry (LEARN-165/LEARN-222).
//
// One small interface — Enable / Disable / Status — hides three per-platform
// adapters behind build tags: the macOS launchd plist, the Linux XDG
// autostart desktop file, and the Windows Startup-folder shortcut (.lnk
// written through a PowerShell WScript.Shell one-liner).  The interface is
// the test surface: a Manager constructed with an explicit EntryPath writes
// and reads real files in a temp dir, with no mocks.
//
// The entry runs the same daemon command line everywhere —
// `pharos start --background --no-open --daemon` — plus an explicit `--port`
// pin.  The pin is what makes the entry's port *detectable*: status compares
// the embedded port with the configured port and reports "enabled (stale
// port)" after `pharos config set port N`, and LEG-219's orchestrator can
// report "rewritten for N (was M)".  Without the pin the entry would be
// indistinguishable for any config, and the stale-port signal (the
// LEARN-219/LEARN-165 self-healing hook) would be impossible.
package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Status is the vocabulary shared with `pharos setup` (LEARN-166 #281):
// enabled | enabled (stale port) | disabled | failed.  Stale is a *suffix* —
// consumers treat "enabled (stale port)" as enabled-with-a-fix.
type Status string

const (
	StatusEnabled   Status = "enabled"
	StatusStalePort Status = "enabled (stale port)"
	StatusDisabled  Status = "disabled"
	StatusFailed    Status = "failed"
)

// Report is the status-view of the startup entry.  Port is the port pinned
// in the entry's command line (0 when disabled or unknown).
type Report struct {
	Enabled   bool
	Status    Status
	EntryPath string
	Port      int
}

// Options configures a Manager.  Exe defaults to os.Executable() (real path,
// symlinks resolved — `go install` rewrites ~/go/bin/pharos in place, so the
// pinned path stays valid across updates).  EntryPath defaults to the
// platform's per-user startup entry path.
type Options struct {
	Exe       string
	Port      int
	EntryPath string
}

// Manager writes and inspects one platform's startup entry.
type Manager struct {
	exe       string
	entryPath string
	port      int // configured port, embedded at enable time
}

// New builds a Manager for the current platform.  It fails when the platform
// has no supported startup entry (EntryPath must then be provided explicitly
// — used by tests, which point it into a temp dir).
func New(opts Options) (*Manager, error) {
	exe := opts.Exe
	if exe == "" {
		p, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable: %w", err)
		}
		exe = p
		if rp, err := filepath.EvalSymlinks(exe); err == nil {
			exe = rp
		}
	}
	entry := opts.EntryPath
	if entry == "" {
		e, err := entryPath()
		if err != nil {
			return nil, err
		}
		entry = e
	}
	return &Manager{exe: exe, entryPath: entry, port: opts.Port}, nil
}

// EntryPath is where this manager's startup entry lives (or would live).
func (m *Manager) EntryPath() string { return m.entryPath }

// args is the daemon command line the entry runs: the same path a user runs
// manually (`pharos start --background --no-open --daemon`), plus the
// explicit --port pin (see package comment).
func args(port int) []string {
	return []string{"start", "--background", "--no-open", "--daemon", "--port", strconv.Itoa(port)}
}

// portFromArgs extracts the --port value from a daemon command line,
// 0 when absent or malformed (unknown port = not stale, and enable is the
// fix either way).
func portFromArgs(a []string) int {
	for i := 0; i+1 < len(a); i++ {
		if a[i] == "--port" {
			p, err := strconv.Atoi(a[i+1])
			if err != nil {
				return 0
			}
			return p
		}
	}
	return 0
}

// Enable writes (or rewrites) the startup entry for the manager's port and
// returns the port the previous entry was pinned to (0 when there was none
// or it couldn't be read).  Rewriting an enabled entry is self-healing
// (LEARN-165 #261 Q3): it embeds the current configured port, so a later
// `status` reports the correct port.
func (m *Manager) Enable() (int, error) {
	prev, _ := m.readPort()
	if err := m.write(args(m.port)); err != nil {
		return prev, err
	}
	return prev, nil
}

// Disable removes the startup entry.  Removing a non-existent entry is a
// no-op success (Unix-y, script-friendly — LEARN-165 #261 Q3).
func (m *Manager) Disable() error {
	err := os.Remove(m.entryPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Status reports whether the entry exists and whether its embedded port
// matches the configured port.
func (m *Manager) Status() Report {
	r := Report{EntryPath: m.entryPath}
	exists, port, _ := m.read()
	if !exists {
		r.Status = StatusDisabled
		return r
	}
	r.Enabled = true
	r.Port = port
	if port != 0 && port != m.port {
		r.Status = StatusStalePort
	} else {
		r.Status = StatusEnabled
	}
	return r
}

// write is the platform-specific entry writer.
func (m *Manager) write(daemonArgs []string) error {
	return m.writeEntry(daemonArgs)
}

// read reports whether the entry exists and the port pinned in its command
// line (0 when the entry exists but the port can't be determined).
func (m *Manager) read() (exists bool, port int, err error) {
	return m.readEntry()
}

// readPort is the best-effort port read used by Enable's self-healing
// rewrite: it never blocks a write.
func (m *Manager) readPort() (int, error) {
	_, port, _ := m.read()
	return port, nil
}
