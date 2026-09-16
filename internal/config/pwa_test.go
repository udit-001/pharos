package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPWAFileRoundtrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := PWAInstalled{Installed: true, Origin: "http://127.0.0.1:9090", Browser: "Edge", At: "2026-09-15T21:00:00Z"}
	if err := WritePWAFile(want); err != nil {
		t.Fatalf("WritePWAFile: %v", err)
	}
	got, err := ReadPWAFile()
	if err != nil {
		t.Fatalf("ReadPWAFile: %v", err)
	}
	if got == nil || *got != want {
		t.Fatalf("ReadPWAFile = %+v, want %+v", got, want)
	}
}

func TestReadPWAFileAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := ReadPWAFile()
	if err != nil || got != nil {
		t.Fatalf("ReadPWAFile = %v, %v; want nil, nil", got, err)
	}
}

func TestReadPWAFileCorruptIsAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(PWAFilePath(), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	got, err := ReadPWAFile()
	if err != nil || got != nil {
		t.Fatalf("ReadPWAFile (corrupt) = %v, %v; want nil, nil", got, err)
	}
}

func TestWritePWAFileAtomicNoTempLeftover(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WritePWAFile(PWAInstalled{Installed: true}); err != nil {
		t.Fatalf("WritePWAFile: %v", err)
	}
	if _, err := os.Stat(PWAFilePath() + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file left behind (err=%v)", err)
	}
	// Overwrite works.
	if err := WritePWAFile(PWAInstalled{Installed: false, Origin: "http://127.0.0.1:9999"}); err != nil {
		t.Fatalf("WritePWAFile (overwrite): %v", err)
	}
	got, _ := ReadPWAFile()
	if got == nil || got.Installed || got.Origin != "http://127.0.0.1:9999" {
		t.Fatalf("overwrite = %+v", got)
	}
	// The file lands in the config dir next to server.pid.
	if filepath.Dir(PWAFilePath()) != ConfigDir() {
		t.Fatalf("PWAFilePath dir = %s, want %s", filepath.Dir(PWAFilePath()), ConfigDir())
	}
}
