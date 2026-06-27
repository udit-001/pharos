package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DataDir string `toml:"data_dir"`
}

func homeDir() string {
	d, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return d
}

func ConfigDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = filepath.Join(homeDir(), ".config")
	}
	return filepath.Join(d, "pharos")
}

func DefaultDataDir() string {
	return filepath.Join(homeDir(), ".pharos")
}

func Path() string {
	return filepath.Join(ConfigDir(), "pharos.toml")
}

func PidPath() string {
	return filepath.Join(ConfigDir(), "server.pid")
}

func Load() (*Config, error) {
	p := Path()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultDataDir()
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if strings.HasPrefix(cfg.DataDir, "~/") {
		cfg.DataDir = filepath.Join(homeDir(), cfg.DataDir[2:])
	} else if cfg.DataDir == "~" {
		cfg.DataDir = homeDir()
	}
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	p := Path()
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("create %s: %w", p, err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
