package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Start-warning tests use non-default ports (never 9090) so they stay
// hermetic on machines where the real Pharos daemon is running.

// TestStartPortMismatchWarning covers LEARN-219 D3: `pharos start` warns when
// the running server's port differs from the configured port.
func TestStartPortMismatchWarning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runningPort, configPort := 19730, 19731
	writeConfigWithPort(t, configPort)
	writePidFile(t, runningPort, 424242)
	defer fakeServer(t, runningPort)()

	out, err := captureCLIResult(t, []string{"start"})
	if !isExitCode(t, err, 1) {
		t.Fatal("already-running start must exit non-zero (LEARN-235)")
	}
	want := fmt.Sprintf("\n  Pharos running on %d (PID %d), config says %d.\n  Fix: pharos setup\n  Dashboard: http://127.0.0.1:%d/\n\n", runningPort, 424242, configPort, runningPort)
	if out != want {
		t.Fatalf("start output = %q, want %q", out, want)
	}
}

// TestStartSamePortKeepsExistingMessage: the mismatch warning must not
// replace the normal already-running output when ports match.
func TestStartSamePortKeepsExistingMessage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19732
	writeConfigWithPort(t, port)
	writePidFile(t, port, 424242)
	defer fakeServer(t, port)()

	out, err := captureCLIResult(t, []string{"start"})
	if !isExitCode(t, err, 1) {
		t.Fatal("already-running start must exit non-zero (LEARN-235)")
	}
	if !strings.Contains(out, "already running") || strings.Contains(out, "config says") {
		t.Fatalf("start output = %q", out)
	}
	if !strings.Contains(out, fmt.Sprintf("http://127.0.0.1:%d/", port)) {
		t.Fatalf("start output missing dashboard URL: %q", out)
	}
}

// TestStartMismatchJSONUnchanged: --json output shape is untouched by the
// console-only warning (no new keys → no consumer breakage).
func TestStartMismatchJSONUnchanged(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runningPort, configPort := 19733, 19734
	writeConfigWithPort(t, configPort)
	writePidFile(t, runningPort, 424242)
	defer fakeServer(t, runningPort)()

	out, err := captureCLIResult(t, []string{"start", "--json"})
	if !isExitCode(t, err, 1) {
		t.Fatal("already-running start must exit non-zero even with --json (LEARN-235)")
	}
	// map-encoded → alphabetical key order.
	want := fmt.Sprintf("{\n  \"port\": %d,\n  \"running\": true,\n  \"url\": \"http://127.0.0.1:%d/\"\n}\n", runningPort, runningPort)
	if out != want {
		t.Fatalf("start --json = %q, want %q", out, want)
	}
}

// LEARN-235: a second `pharos start` while a server runs must exit
// non-zero with a plain-language message naming the running instance —
// a loud refusal, not a silent no-op.
func TestStartAlreadyRunningExitsNonZero(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19735
	writeConfigWithPort(t, port)
	writePidFile(t, port, 424242)
	defer fakeServer(t, port)()

	out, err := captureCLIResult(t, []string{"start"})
	if err == nil {
		t.Fatal("second start while running must exit non-zero")
	}
	var ee exitError
	if !errors.As(err, &ee) || ee.code != 1 {
		t.Fatalf("err = %v, want exitError(1)", err)
	}
	if !strings.Contains(out, "already running") || !strings.Contains(out, "424242") {
		t.Fatalf("refusal must name the running instance (PID), got: %q", out)
	}
}

func TestStartAlreadyRunningJSONExitsNonZero(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19736
	writeConfigWithPort(t, port)
	writePidFile(t, port, 424242)
	defer fakeServer(t, port)()

	out, err := captureCLIResult(t, []string{"start", "--json"})
	if err == nil {
		t.Fatal("second start while running must exit non-zero even with --json")
	}
	// JSON shape unchanged (consumers parse it; only the exit code changes).
	want := fmt.Sprintf("{\n  \"port\": %d,\n  \"running\": true,\n  \"url\": \"http://127.0.0.1:%d/\"\n}\n", port, port)
	if out != want {
		t.Fatalf("start --json = %q, want %q", out, want)
	}
}
