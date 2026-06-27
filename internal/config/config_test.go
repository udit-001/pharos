package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg := &Config{DataDir: "/custom/data/path"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DataDir != "/custom/data/path" {
		t.Fatalf("expected /custom/data/path, got %s", got.DataDir)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load for missing file: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil, got %+v", cfg)
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "pharos")
	os.MkdirAll(cfgDir, 0755)
	os.WriteFile(filepath.Join(cfgDir, "pharos.toml"), []byte("not valid toml {{{"), 0644)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for corrupt config")
	}
}

func TestDefaultDataDir(t *testing.T) {
	dd := DefaultDataDir()
	if dd == "" {
		t.Fatal("DefaultDataDir() returned empty")
	}
	if !filepath.IsAbs(dd) {
		t.Fatalf("DefaultDataDir() should be absolute: %s", dd)
	}
}

func TestSaveExpandsTilde(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	home, _ := os.UserHomeDir()
	cfg := &Config{DataDir: "~/my-pharos"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if cfg.DataDir != filepath.Join(home, "my-pharos") {
		t.Fatalf("expected %s, got %s", filepath.Join(home, "my-pharos"), cfg.DataDir)
	}
}

func TestPaths(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	p := Path()
	if !filepath.IsAbs(p) {
		t.Fatalf("Path() should be absolute: %s", p)
	}
	if filepath.Base(p) != "pharos.toml" {
		t.Fatalf("Path() should end with pharos.toml: %s", p)
	}

	pp := PidPath()
	if filepath.Base(pp) != "server.pid" {
		t.Fatalf("PidPath() should end with server.pid: %s", pp)
	}

	cd := ConfigDir()
	if filepath.Base(cd) != "pharos" {
		t.Fatalf("ConfigDir() should end with pharos: %s", cd)
	}
}
