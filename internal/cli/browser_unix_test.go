//go:build !windows

package cli

import (
	"errors"
	"runtime"
	"testing"
)

// Tier-1 ladder tests are !windows: they reference chromiumCandidates
// (unix-only) and a stubbed run on the Windows tier would otherwise drive
// openSetupPage into the real createURLLauncher, writing Pharos.url to the
// user's Desktop during `go test`.

func TestChromiumCandidatesOrder(t *testing.T) {
	names := chromiumCandidates()
	if len(names) < 7 {
		t.Fatalf("expected >=7 candidates, got %d", len(names))
	}
	// Preference order: Chrome → Edge → Brave → Chromium → helium (LEARN-170).
	pref := []string{"Chrome", "Edge", "Brave", "Chromium", "helium"}
	seen := make(map[string]int)
	for i, c := range names {
		if seen[c.name] == 0 {
			seen[c.name] = i
		}
	}
	last := -1
	for _, n := range pref {
		idx, ok := seen[n]
		if !ok {
			t.Fatalf("candidate %s missing", n)
		}
		if idx < last {
			t.Fatalf("preference order violated: %s at %d after %d", n, idx, last)
		}
		last = idx
	}
	if runtime.GOOS == "linux" && names[0].bin != "google-chrome" {
		t.Fatalf("linux tier-1 = %+v, want google-chrome first", names[0])
	}
}

func TestLadderTier1Chromium(t *testing.T) {
	run := func(name string, args ...string) error { return nil }
	var opened []string
	openDefault := func(url string) error {
		opened = append(opened, url)
		return nil
	}
	res := openSetupPage(9090, stubLookPath("google-chrome"), run, openDefault)
	if !res.opened || res.method != "chromium" || res.name != "Chrome" || res.url != "http://127.0.0.1:9090/setup" {
		t.Fatalf("tier1 = %+v", res)
	}
	if len(opened) != 0 {
		t.Fatalf("default browser should not be tried on tier-1 success: %v", opened)
	}
}

func TestLadderTier2DefaultBrowser(t *testing.T) {
	run := func(name string, args ...string) error { return errors.New("launch failed") }
	res := openSetupPage(9090, stubLookPath("google-chrome"), run, func(url string) error { return nil })
	if !res.opened || res.method != "default" || res.name != "default browser" {
		t.Fatalf("tier2 = %+v", res)
	}
}

func TestLadderTier3PrintURL(t *testing.T) {
	run := func(name string, args ...string) error { return errors.New("launch failed") }
	res := openSetupPage(9090, stubLookPath("google-chrome"), run, func(url string) error { return errors.New("no default") })
	if res.opened || res.method != "none" || res.shortcutCreated {
		t.Fatalf("tier3 = %+v", res)
	}
	if res.url != "http://127.0.0.1:9090/setup" {
		t.Fatalf("tier3 url = %s", res.url)
	}
}

func TestLadderZeroChromiumManualOpen(t *testing.T) {
	// No candidate detected (Firefox-only) → no page, no default open.
	run := func(name string, args ...string) error { return nil }
	openDefault := func(url string) error {
		t.Fatalf("default browser must not open on a Firefox-only machine")
		return nil
	}
	res := openSetupPage(9090, stubLookPath(), run, openDefault)
	if res.opened || res.method != "none" || res.shortcutCreated {
		t.Fatalf("firefox-only = %+v", res)
	}
	if res.url != "http://127.0.0.1:9090/setup" {
		t.Fatalf("no-page url = %s", res.url)
	}
}
