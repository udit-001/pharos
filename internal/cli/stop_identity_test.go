//go:build !windows

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/udit-001/pharos/internal/config"
)

// LEARN-235: a stale or reused pidfile must not let `pharos stop` signal an
// arbitrary process. On unix, os.FindProcess succeeds for any pid — so stop
// must verify the pid actually belongs to a pharos process before signaling.

func TestStopForeignPIDNotSignaled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// A live process that is not pharos: sleep, with a lifetime bound so a
	// test failure cannot leak a stray process.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("spawn sleep: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	writePidFile(t, 9090, cmd.Process.Pid)

	if outcome := stopServerByPidfile(); outcome != stopForeignPID {
		t.Fatalf("outcome = %v, want stopForeignPID", outcome)
	}

	// The foreign process must have survived the refused stop: a signal-0
	// probe still finds it alive.
	if err := syscall.Kill(cmd.Process.Pid, syscall.Signal(0)); err != nil {
		t.Fatalf("sleep must survive the refused stop: %v", err)
	}
	if _, err := os.Stat(config.PidPath()); !os.IsNotExist(err) {
		t.Fatal("stale pidfile must be cleaned up even on refusal")
	}
}

// TestStopForeignPIDTestBinary: the test binary itself is a live non-pharos
// process — a pidfile naming it must be refused too.
func TestStopForeignPIDTestBinary(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writePidFile(t, 9090, os.Getpid())

	if outcome := stopServerByPidfile(); outcome != stopForeignPID {
		t.Fatalf("outcome = %v, want stopForeignPID (test binary is not pharos)", outcome)
	}
}

func TestStopDeadPIDBecomesStale(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writePidFile(t, 9090, 999999999)
	if outcome := stopServerByPidfile(); outcome != stopStalePID {
		t.Fatalf("outcome = %v, want stopStalePID", outcome)
	}
	if _, err := os.Stat(config.PidPath()); !os.IsNotExist(err) {
		t.Fatal("stale pidfile must be cleaned up")
	}
}

// TestStopJSONForeignPID pins the JSON contract of the refusal.
func TestStopJSONForeignPID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writePidFile(t, 9090, os.Getpid())

	out := captureCLI(t, []string{"stop", "--json"})
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if m["running"] != false {
		t.Fatalf("running = %v, want false", m["running"])
	}
	msg, _ := m["message"].(string)
	if msg == "" {
		t.Fatalf("message missing in %q", out)
	}
}

// TestProcessNameProbes pins the identity probe itself.
func TestProcessNameProbes(t *testing.T) {
	self := processName(os.Getpid())
	if self == "" {
		t.Fatal("processName must resolve the test process")
	}
	// Test binaries are "cli.test"-shaped, never bare "pharos".
	if self == "pharos" {
		t.Fatalf("unexpected pharos match: %q", self)
	}
	if processName(999999999) != "" {
		t.Fatal("impossible pid must not resolve")
	}
	if processIsPharos(999999999) {
		t.Fatal("impossible pid must not count as pharos")
	}
}
