//go:build windows

package cli

import (
	"os"
	"path/filepath"
)

// chromiumTier is the Windows side of tier 1: Edge is always the preferred
// Chromium (ships on Win10/11, LEARN-170 Q2), launched via ShellExecute
// app-id `start msedge <url>` (Q4 decision (a)). When the msedge launch
// fails, Windows cannot distinguish "Edge uninstalled, Firefox is the
// default" from "Edge uninstalled, Chrome is the default" — the handoff
// decision is to treat it as the Firefox-only case (LEARN-170 #274): never
// open /setup in an unknown browser, create the launcher instead.
func chromiumTier(port int, _ lookPathFunc, run execRunner) (name string, opened bool, found bool) {
	launch := msedgeLaunch(setupURL(port))
	if err := run(launch[0], launch[1:]...); err == nil {
		return "Edge", true, true
	}
	return "", false, false
}

// createURLLauncher writes Pharos.url (Internet Shortcut, INI) on the
// Desktop — the PWA-less fallback for a Windows box without Edge, pinnable
// to the taskbar/Start (LEARN-170 #274, Windows-first).
func createURLLauncher(port int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "Pharos.url")
	if err := os.WriteFile(path, urlShortcutContent(dashboardURLFor(port)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
