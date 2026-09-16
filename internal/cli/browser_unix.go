//go:build !windows

package cli

import (
	"os"
	"runtime"
)

// browserCandidate is one tier-1 Chromium entry. appDir identifies a macOS
// app bundle (launched via `open -a <name>`); bin is a PATH binary (launched
// directly with the URL argument, no --app).
type browserCandidate struct {
	name   string
	bin    string
	appDir string
}

// chromiumCandidates returns the tier-1 search order (LEARN-170 Q2):
// macOS/Linux = Chrome → Edge → Brave → Chromium → helium.
func chromiumCandidates() []browserCandidate {
	if runtime.GOOS == "darwin" {
		return []browserCandidate{
			{"Chrome", "", "/Applications/Google Chrome.app"},
			{"Edge", "", "/Applications/Microsoft Edge.app"},
			{"Brave", "", "/Applications/Brave Browser.app"},
			{"Chromium", "", "/Applications/Chromium.app"},
			{"helium", "helium", ""},
		}
	}
	return []browserCandidate{
		{"Chrome", "google-chrome", ""},
		{"Chrome", "google-chrome-stable", ""},
		{"Edge", "microsoft-edge", ""},
		{"Brave", "brave-browser", ""},
		{"Chromium", "chromium", ""},
		{"Chromium", "chromium-browser", ""},
		{"helium", "helium", ""},
	}
}

func candidatePresent(c browserCandidate, lp lookPathFunc) bool {
	if c.appDir != "" {
		_, err := os.Stat(c.appDir)
		return err == nil
	}
	_, err := lp(c.bin)
	return err == nil
}

func launchCandidate(c browserCandidate, url string, lp lookPathFunc, run execRunner) bool {
	if c.appDir != "" {
		return run("open", "-a", c.name, url) == nil
	}
	bin, err := lp(c.bin)
	if err != nil {
		return false
	}
	return run(bin, url) == nil
}

// chromiumTier is the macOS/Linux side of tier 1: detect the first available
// Chromium in preference order, then chain down the binaries until one
// launches. found=false means a Firefox-only machine (zero Chromium); the
// caller then skips tier 2 entirely (LEARN-170 #274).
func chromiumTier(port int, lp lookPathFunc, run execRunner) (name string, opened bool, found bool) {
	url := setupURL(port)
	present := false
	for _, c := range chromiumCandidates() {
		if !candidatePresent(c, lp) {
			continue
		}
		present = true
		if launchCandidate(c, url, lp, run) {
			return c.name, true, true
		}
	}
	if !present {
		return "", false, false
	}
	return "", false, true
}

// createURLLauncher is a later pass on macOS/Linux (LEARN-170 #274:
// .webloc/.desktop equivalents) — nothing is written this cycle.
func createURLLauncher(port int) (string, error) {
	return "", nil
}
