package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/config"
)

// decodeAutostartJSON unmarshals a --json output into a map so tests can
// assert both the exact key set and the values.
func decodeAutostartJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("unmarshal %q: %v", out, err)
	}
	return m
}

// autostart CLI tests drive the real cobra commands against a temp
// XDG_CONFIG_HOME (the Linux entry lives under $XDG_CONFIG_HOME/autostart),
// so every file the commands touch is real — no mocks, no HOME pollution.

func writeConfigWithPort(t *testing.T, port int) {
	t.Helper()
	if err := config.Save(&config.Config{Port: port}); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

// configDirFor returns the temp XDG_CONFIG_HOME (the Linux autostart entry
// lives directly under it).
func configDirFor(t *testing.T) string {
	return os.Getenv("XDG_CONFIG_HOME")
}

// captureCLI runs real cobra commands against the real (temp-dir) config,
// capturing stdout. The autostart/setup commands skip the DB via root's
// PersistentPreRunE, so with XDG_CONFIG_HOME set to a temp dir nothing
// touches the developer's HOME.
func captureCLI(t *testing.T, args []string) string {
	t.Helper()
	root := newRootForTest()
	root.SetArgs(args)
	return captureStdout(t, func() error { return root.Execute() })
}

func TestAutostartEnableJSONContract(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	out := strings.TrimSpace(captureCLI(t, []string{"autostart", "enable", "--json"}))
	m := decodeAutostartJSON(t, out)
	want := map[string]any{"enabled": true, "status": "enabled", "entry_path": filepath.Join(configDirFor(t), "autostart", "pharos.desktop"), "port": float64(9090)}
	if len(m) != len(want) {
		t.Fatalf("key set = %d, want %d (%s)", len(m), len(want), out)
	}
	for k, v := range want {
		if m[k] != v {
			t.Fatalf("%s = %v, want %v (%s)", k, m[k], v, out)
		}
	}
	// Key order is part of the contract — a struct keeps it stable.
	lines := strings.Split(out, "\n")
	if len(lines) < 3 || !strings.Contains(lines[1], `"enabled"`) || !strings.Contains(lines[2], `"status"`) {
		t.Fatalf("unexpected key order: %s", out)
	}
	if _, err := os.Stat(filepath.Join(configDirFor(t), "autostart", "pharos.desktop")); err != nil {
		t.Fatalf("entry not written: %v", err)
	}
}

func TestAutostartStatusJSONContract(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	base := configDirFor(t)

	// Disabled: port 0, entry_path is the would-be path.
	out := decodeAutostartJSON(t, captureCLI(t, []string{"autostart", "status", "--json"}))
	wantDisabled := map[string]any{"enabled": false, "status": "disabled", "entry_path": filepath.Join(base, "autostart", "pharos.desktop"), "port": float64(0)}
	for k, v := range wantDisabled {
		if out[k] != v {
			t.Fatalf("status --json (disabled) %s = %v, want %v", k, out[k], v)
		}
	}

	// Enabled: port from the entry.
	captureCLI(t, []string{"autostart", "enable"})
	writeConfigWithPort(t, 9090)
	out = decodeAutostartJSON(t, captureCLI(t, []string{"autostart", "status", "--json"}))
	wantEnabled := map[string]any{"enabled": true, "status": "enabled", "entry_path": filepath.Join(base, "autostart", "pharos.desktop"), "port": float64(9090)}
	for k, v := range wantEnabled {
		if out[k] != v {
			t.Fatalf("status --json (enabled) %s = %v, want %v", k, out[k], v)
		}
	}

	// Stale port: config moved to 9091, entry still pinned 9090.
	writeConfigWithPort(t, 9091)
	out = decodeAutostartJSON(t, captureCLI(t, []string{"autostart", "status", "--json"}))
	wantStale := map[string]any{"enabled": true, "status": "enabled (stale port)", "entry_path": filepath.Join(base, "autostart", "pharos.desktop"), "port": float64(9090)}
	for k, v := range wantStale {
		if out[k] != v {
			t.Fatalf("status --json (stale) %s = %v, want %v", k, out[k], v)
		}
	}
}

func TestAutostartEnableRewritesStaleEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)
	captureCLI(t, []string{"autostart", "enable"})

	// Port changed; enable rewrites and the human output shows the tail.
	writeConfigWithPort(t, 9091)
	out := captureCLI(t, []string{"autostart", "enable"})
	if !strings.Contains(out, "✓ Autostart enabled") || !strings.Contains(out, "Port:  9091 (was 9090)") {
		t.Fatalf("enable rewrite output missing rewrite note:\n%s", out)
	}
	r := decodeAutostartJSON(t, captureCLI(t, []string{"autostart", "status", "--json"}))
	if r["status"] != "enabled" || r["port"] != float64(9091) {
		t.Fatalf("entry not rewritten to 9091:\n%v", r)
	}
}

func TestAutostartDisable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	base := configDirFor(t)
	writeConfigWithPort(t, 9090)
	captureCLI(t, []string{"autostart", "enable"})

	out := decodeAutostartJSON(t, captureCLI(t, []string{"autostart", "disable", "--json"}))
	want := map[string]any{"enabled": false, "status": "disabled", "entry_path": filepath.Join(base, "autostart", "pharos.desktop"), "port": float64(0)}
	for k, v := range want {
		if out[k] != v {
			t.Fatalf("disable --json %s = %v, want %v", k, out[k], v)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "autostart", "pharos.desktop")); !os.IsNotExist(err) {
		t.Fatal("entry still present after disable")
	}
	// Disabling again is a no-op success (script-friendly).
	captureCLI(t, []string{"autostart", "disable"})
}

func TestAutostartHumanDisableOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)
	captureCLI(t, []string{"autostart", "enable"})
	out := captureCLI(t, []string{"autostart", "disable"})
	if !strings.Contains(out, "✓ Autostart disabled") {
		t.Fatalf("disable output missing confirmation:\n%s", out)
	}
}

// TestAutostartLeavesSkipDB guards the root.go skip-list contract: the
// autostart family is a pure config-file operation and must never open or
// create the data DB (which lives under HOME/.pharos even when XDG is
// redirected). A regression here means PersistentPreRunE opened the DB.
func TestAutostartLeavesSkipDB(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9091)

	t.Run("enable", func(t *testing.T) { captureCLI(t, []string{"autostart", "enable"}) })
	t.Run("status", func(t *testing.T) { captureCLI(t, []string{"autostart", "status"}) })
	t.Run("disable", func(t *testing.T) { captureCLI(t, []string{"autostart", "disable"}) })

	if _, err := os.Stat(filepath.Join(home, ".pharos", "pharos.db")); !os.IsNotExist(err) {
		t.Fatalf("DB created under HOME by autostart command (err=%v) — skip list regression", err)
	}
}

func TestAutostartStatusHumanStaleShowsFix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)
	captureCLI(t, []string{"autostart", "enable"})
	writeConfigWithPort(t, 9091)
	out := captureCLI(t, []string{"autostart", "status"})
	if !strings.Contains(out, "Autostart: enabled (stale port)") ||
		!strings.Contains(out, "Fix:      pharos autostart enable") {
		t.Fatalf("human stale status missing vocabulary/fix:\n%s", out)
	}
}
