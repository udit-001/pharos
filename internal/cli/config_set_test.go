package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/autostart"
	"github.com/udit-001/pharos/internal/config"
)

func TestConfigSetPortNewValue(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	out := captureCLI(t, []string{"config", "set", "port", "9091"})
	want := "\n  OK: port 9091\n  PWA pinned to 9090 is now stale.\n  Fix: pharos setup\n\n"
	if out != want {
		t.Fatalf("set port output = %q, want %q", out, want)
	}
	// Config actually persisted.
	cfg, err := config.Load()
	if err != nil || cfg.Port != 9091 {
		t.Fatalf("cfg.Port = %d (err=%v), want 9091", cfg.Port, err)
	}
}

func TestConfigSetPortSameValueNoStaleWarning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	out := captureCLI(t, []string{"config", "set", "port", "9090"})
	want := "\n  OK: port 9090\n\n"
	if out != want {
		t.Fatalf("same-port output = %q, want %q", out, want)
	}
}

func TestConfigSetPortHealsAutostartEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	// Real entry pinned at the old port.
	am, err := autostart.New(autostart.Options{Port: 9090})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := am.Enable(); err != nil {
		t.Fatal(err)
	}

	out := captureCLI(t, []string{"config", "set", "port", "9091"})
	// Silent heal: the output must not mention autostart at all.
	if strings.Contains(out, "autostart") {
		t.Fatalf("heal must be silent, output = %q", out)
	}
	entry := filepath.Join(configDirFor(t), "autostart", "pharos.desktop")
	data, err := os.ReadFile(entry)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if !strings.Contains(string(data), "--port 9091") {
		t.Fatalf("entry not rewritten to 9091:\n%s", data)
	}
	if strings.Contains(string(data), "--port 9090") {
		t.Fatalf("entry still pind the old port:\n%s", data)
	}
}

func TestConfigSetPortDoesNotResurrectDisabled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	// Enable then disable — the entry file should be removed.
	captureCLI(t, []string{"autostart", "enable"})
	disabledOut := captureCLI(t, []string{"autostart", "disable"})
	if !strings.Contains(disabledOut, "disabled") {
		t.Fatalf("disable output = %q", disabledOut)
	}

	// Now change port — must NOT resurrect the disabled entry.
	out := captureCLI(t, []string{"config", "set", "port", "9091"})
	if !strings.Contains(out, "OK: port 9091") {
		t.Fatalf("output = %q", out)
	}
	// Entry was deleted by disable; heal must not recreate it.
	entry := filepath.Join(configDirFor(t), "autostart", "pharos.desktop")
	if _, err := os.Stat(entry); !os.IsNotExist(err) {
		t.Fatal("port change must not resurrect a disabled autostart entry")
	}
}

func TestConfigSetDataDirUntouched(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeConfigWithPort(t, 9090)

	out := captureCLI(t, []string{"config", "set", "data_dir", "/tmp/nonexistent-test-dir"})
	if !strings.Contains(out, "✓ data_dir set to /tmp/nonexistent-test-dir") {
		t.Fatalf("data_dir output = %q", out)
	}
	if !strings.Contains(out, "Config:") {
		t.Fatalf("data_dir output must keep the Config path line: %q", out)
	}
}
