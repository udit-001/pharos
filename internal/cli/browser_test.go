package cli

import (
	"errors"
	"testing"
)

// Platform-neutral ladder-building blocks. The tier-1 ladder itself is
// exercised in browser_unix_test.go (`!windows`): its tests reference
// chromiumCandidates (unix-only) and the Windows tier-1 failure path would
// write a real Pharos.url to the Desktop.

func TestMsedgeLaunch(t *testing.T) {
	got := msedgeLaunch("http://127.0.0.1:9090/setup")
	want := []string{"cmd", "/c", "start", "msedge", "http://127.0.0.1:9090/setup"}
	if len(got) != len(want) {
		t.Fatalf("msedgeLaunch = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("msedgeLaunch = %v, want %v", got, want)
		}
	}
}

func TestURLShortcutContent(t *testing.T) {
	got := string(urlShortcutContent("http://127.0.0.1:9090/"))
	want := "[InternetShortcut]\r\nURL=http://127.0.0.1:9090/\r\n"
	if got != want {
		t.Fatalf("urlShortcutContent = %q, want %q", got, want)
	}
}

func TestURLs(t *testing.T) {
	if got := setupURL(9090); got != "http://127.0.0.1:9090/setup" {
		t.Fatalf("setupURL = %s", got)
	}
	if got := dashboardURLFor(9090); got != "http://127.0.0.1:9090/" {
		t.Fatalf("dashboardURLFor = %s", got)
	}
}

// stubLookPath returns a lookup func where present lists resolve and others
// fail (no real PATH dependence — deterministic ladder tests).
func stubLookPath(present ...string) lookPathFunc {
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	return func(file string) (string, error) {
		if set[file] {
			return "/usr/bin/" + file, nil
		}
		return "", errNotFound
	}
}

var errNotFound = errors.New("not found")
