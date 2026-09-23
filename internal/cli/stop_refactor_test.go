package cli

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"github.com/udit-001/pharos/internal/config"
)

func TestStopNoServerJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out := captureCLI(t, []string{"stop", "--json"})
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if m["running"] != false || m["message"] != "no server running" {
		t.Fatalf("stop --json = %v", m)
	}
}

// TestStopServerByPidfileOutcomes locks the stopOutcome mapping so the
// `pharos stop` command and `pharos setup`'s port-change path share the
// same verdicts (LEARN-221 extraction).
func TestStopServerByPidfileOutcomes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// No pid file → no server.
	if outcome := stopServerByPidfile(); outcome != stopNoServer {
		t.Fatalf("no pidfile: outcome = %v, want stopNoServer", outcome)
	}

	// A pidfile naming a dead process → stopStalePID (liveness probe first,
	// LEARN-235) + cleaned up.
	writePidFile(t, 9090, 999999999)
	if outcome := stopServerByPidfile(); outcome != stopStalePID {
		t.Fatalf("dead-pid pidfile: outcome = %v, want stopStalePID", outcome)
	}
	if _, err := os.Stat(config.PidPath()); !os.IsNotExist(err) {
		t.Fatal("pidfile must have been cleaned up")
	}
}

func TestProcessAlive(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("own pid must be alive")
	}
	if runtime.GOOS == "windows" {
		t.Skip("windows no-op probe")
	}
	if processAlive(999999999) {
		t.Fatal("impossible pid must be dead")
	}
}
