package cli

import (
	"fmt"
	"os"
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
	writePidFile(t, runningPort, os.Getpid())
	defer fakeServer(t, runningPort)()

	out := captureCLI(t, []string{"start"})
	want := fmt.Sprintf("\n  Pharos running on %d, config says %d.\n  Fix: pharos setup\n  Dashboard: http://127.0.0.1:%d/\n\n", runningPort, configPort, runningPort)
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
	writePidFile(t, port, os.Getpid())
	defer fakeServer(t, port)()

	out := captureCLI(t, []string{"start"})
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
	writePidFile(t, runningPort, os.Getpid())
	defer fakeServer(t, runningPort)()

	out := captureCLI(t, []string{"start", "--json"})
	// map-encoded → alphabetical key order.
	want := fmt.Sprintf("{\n  \"port\": %d,\n  \"running\": true,\n  \"url\": \"http://127.0.0.1:%d/\"\n}\n", runningPort, runningPort)
	if out != want {
		t.Fatalf("start --json = %q, want %q", out, want)
	}
}
