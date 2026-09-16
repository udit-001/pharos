package cli

import (
	"fmt"
	"os/exec"

	"github.com/udit-001/pharos/internal/server"
)

// Browser ladder for `pharos setup` (LEARN-170, LEARN-166 #281):
//
//	tier 1 — preferred Chromium: Windows = Edge (always, via ShellExecute
//	        app-id `start msedge <url>`); macOS/Linux = Chrome → Edge → Brave
//	        → Chromium → helium, first detectable wins, per-binary chain-down.
//	        Normal tab, never --app (Q1: app-mode windows report standalone
//	        without installing, corrupting /setup's detection state machine).
//	tier 2 — default browser (server.OpenBrowser) — only on Chromium-default
//	        systems where every tier-1 binary failed to launch.
//	tier 3 — print the URL for manual open.
//
// Firefox-only machines (zero Chromium) NEVER get /setup opened in a
// non-Chromium browser (LEARN-170 #274): the launcher is created instead
// (Pharos.url on the Windows Desktop; mac/Linux equivalents are a later
// pass) and the outcome printed.

// execRunner is the process-launch seam (default: start without waiting —
// the ladder must not block on a browser window closing).
type execRunner func(name string, args ...string) error

// lookPathFunc is the PATH-lookup seam (default: exec.LookPath).
type lookPathFunc func(file string) (string, error)

// browserResult describes what (if anything) opening /setup did.
type browserResult struct {
	opened          bool
	method          string // chromium | default | none
	name            string // human browser name ("Edge", "Chrome", ...)
	shortcutCreated bool   // Firefox-only launcher written
	url             string // the URL shown to the user
}

func setupURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/setup", port)
}

func dashboardURLFor(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/", port)
}

// msedgeLaunch is the Windows tier-1 launch vector (LEARN-170 Q4 decision
// (a)): ShellExecute via the msedge app-id, cmd /c start — the same pattern
// as the default-browser helper so tier 2 shares the mechanism.
func msedgeLaunch(url string) []string {
	return []string{"cmd", "/c", "start", "msedge", url}
}

// urlShortcutContent renders a Windows Internet Shortcut (.url) — the
// PWA-less fallback launcher for Firefox-only machines (LEARN-170 #274;
// user-pinnable to the taskbar).
func urlShortcutContent(url string) []byte {
	return []byte("[InternetShortcut]\r\nURL=" + url + "\r\n")
}

// openSetupPage drives the ladder for the current platform, per the seam
// functions supplied (recorded browser launches are the stubbed side in
// tests; everything else is real).
func openSetupPage(port int, lp lookPathFunc, run execRunner, openDefault func(url string) error) browserResult {
	url := setupURL(port)
	name, opened, found := chromiumTier(port, lp, run)
	if found && opened {
		return browserResult{opened: true, method: "chromium", name: name, url: url}
	}
	if !found {
		// Firefox-only: never open /setup in a non-Chromium browser
		// (LEARN-170 #274). Launcher on Windows-first this cycle; mac/Linux
		// print the manual-open line instead.
		if path, err := createURLLauncher(port); err == nil && path != "" {
			return browserResult{method: "none", shortcutCreated: true, url: dashboardURLFor(port)}
		}
		return browserResult{method: "none", url: url}
	}
	// Chromium present but every binary failed to launch → default browser.
	if err := openDefault(url); err == nil {
		return browserResult{opened: true, method: "default", name: "default browser", url: url}
	}
	// tier 3: print the URL for manual open.
	return browserResult{method: "none", url: url}
}

// openSetupPageDefault is the real default for the setup runner: real PATH
// lookup, real exec (spawn without waiting), real default-browser open.
func openSetupPageDefault(port int) browserResult {
	return openSetupPage(port, exec.LookPath, runExec, server.OpenBrowser)
}

// runExec spawns a command without waiting for it to exit.
func runExec(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}
