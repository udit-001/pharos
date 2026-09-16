package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PWAInstalled is the pwa.json record written when the dashboard PWA is
// installed from /setup (LEARN-220) and read by `pharos setup` (LEARN-221)
// to skip opening the install page again for the same origin.
type PWAInstalled struct {
	Installed bool   `json:"installed"`
	Origin    string `json:"origin"`  // e.g. http://127.0.0.1:9090
	Browser   string `json:"browser"` // desktop Chrome/Edge/Brave/Chromium, iOS Safari, ...
	At        string `json:"at"`      // RFC3339 install time
}

// PWAFilePath returns the pwa.json record path (next to server.pid).
func PWAFilePath() string {
	return filepath.Join(ConfigDir(), "pwa.json")
}

// ReadPWAFile returns the record, or nil when absent or unreadable — callers
// treat "no record" as "not installed", so a corrupt file must never break
// `pharos setup` (it re-opens the install page, which self-heals).
func ReadPWAFile() (*PWAInstalled, error) {
	data, err := os.ReadFile(PWAFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var r PWAInstalled
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, nil // corrupt record = absent record
	}
	return &r, nil
}

// WritePWAFile atomically writes the record (temp + rename): the server
// handler and `pharos setup` must never observe a half-written pwa.json.
func WritePWAFile(r PWAInstalled) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := PWAFilePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, PWAFilePath())
}
