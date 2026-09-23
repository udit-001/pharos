package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// PidInfo is the server.pid file content: the daemon's process id and the
// port it serves. The pidfile is the contract between the CLI and the
// daemon (LEARN-166): `pharos stop`, `setup`, `notify`, and the server
// lock's refusal message all read it through here.
type PidInfo struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

// ReadPidFile parses the server.pid file. Missing or malformed → error;
// callers treat every error as "no server known".
func ReadPidFile() (*PidInfo, error) {
	data, err := os.ReadFile(PidPath())
	if err != nil {
		return nil, err
	}
	var info PidInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse %s: %w", PidPath(), err)
	}
	return &info, nil
}
