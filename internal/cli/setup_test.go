package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/autostart"
	"github.com/udit-001/pharos/internal/config"
)

// fakeServer listens on a specific 127.0.0.1 port so the real
// isServerRunning health check sees a live server (no real daemons spawned).
// Ports avoid the developer's default 9090 so the suite stays hermetic on
// machines where the real Pharos daemon runs.
func fakeServer(t *testing.T, port int) (shutdown func()) {
	t.Helper()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("listen %d: %v", port, err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}
	go srv.Serve(ln)
	return func() { srv.Close() }
}

func writePidFile(t *testing.T, port, pid int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.PidPath()), 0o755); err != nil {
		t.Fatalf("mkdir pid dir: %v", err)
	}
	// Same shape as the real daemon: JSON {port, pid} (start.go readPidFile).
	data, err := json.Marshal(map[string]int{"port": port, "pid": pid})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.PidPath(), data, 0o644); err != nil {
		t.Fatalf("write pidfile: %v", err)
	}
}

func writePWAInstalled(t *testing.T, port int) {
	t.Helper()
	origin := dashboardURLFor(port)
	origin = strings.TrimSuffix(origin, "/")
	if err := config.WritePWAFile(config.PWAInstalled{Installed: true, Origin: origin, Browser: "Edge", At: "2026-09-15T21:00:00Z"}); err != nil {
		t.Fatalf("write pwa.json: %v", err)
	}
}

// chromiumOpener stubs the tier-1 open with a recorded successful launch.
func chromiumOpener(port int) browserResult {
	return browserResult{opened: true, method: "chromium", name: "Edge", url: setupURL(port)}
}

func TestSetupFreshMachine(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19710
	writeConfigWithPort(t, port)

	var started []int
	var killed bool
	rep, ok := runSetup(port, setupRunner{
		kill:     func() error { killed = true; return nil },
		start:    func(p int) (*exec.Cmd, error) { started = append(started, p); return nil, nil },
		serverUp: func(p int) bool { return true },
		openPWA:  chromiumOpener,
	})
	if !ok {
		t.Fatal("fresh setup must not gate")
	}
	assertSetupServer(t, rep, "started", port, true)
	if killed {
		t.Fatal("fresh machine must not kill anything")
	}
	if len(started) != 1 || started[0] != port {
		t.Fatalf("started = %v, want [%d]", started, port)
	}
	if rep.Autostart.Status != "enabled" || rep.Autostart.PrevPort != 0 {
		t.Fatalf("autostart = %+v", rep.Autostart)
	}
	if !strings.Contains(rep.Autostart.EntryPath, "autostart") {
		t.Fatalf("entry path = %s", rep.Autostart.EntryPath)
	}
	if rep.PWA.Installed || rep.PWA.Origin != fmt.Sprintf("http://127.0.0.1:%d", port) {
		t.Fatalf("pwa = %+v", rep.PWA)
	}
	if !rep.Browser.Opened || rep.Browser.Method != "chromium" || rep.Browser.URL != setupURL(port) {
		t.Fatalf("browser = %+v", rep.Browser)
	}

	// Exact console shape (LEARN-166 #281 fresh mood).
	want := "Server:    started on " + strconv.Itoa(port) + "\n" +
		"Autostart: enabled\n" +
		"PWA:       opening Edge -> " + setupURL(port) + "\n"
	if got := renderSetupHuman(rep); got != want {
		t.Fatalf("render:\n%q\nwant:\n%q", got, want)
	}
}

func TestSetupAlreadySet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19711
	writeConfigWithPort(t, port)
	writePidFile(t, port, os.Getpid())
	defer fakeServer(t, port)()
	writePWAInstalled(t, port)

	rep, ok := runSetup(port, setupRunner{
		kill:     func() error { return nil },
		start:    func(p int) (*exec.Cmd, error) { t.Fatalf("already-set must not start a daemon"); return nil, nil },
		serverUp: func(p int) bool { t.Fatalf("already-set must not poll"); return false },
		openPWA:  func(p int) browserResult { t.Fatalf("already-set must not open the browser"); return browserResult{} },
	})
	if !ok {
		t.Fatal("already-set must not gate")
	}
	assertSetupServer(t, rep, "running", port, true)
	if rep.Autostart.Status != "enabled" {
		t.Fatalf("autostart = %+v (enable rewrites in place; PrevPort == Port)", rep.Autostart)
	}
	if !rep.PWA.Installed || rep.PWA.Origin != fmt.Sprintf("http://127.0.0.1:%d", port) {
		t.Fatalf("pwa = %+v", rep.PWA)
	}
	if rep.Browser.Method != "none" || rep.Browser.URL != dashboardURLFor(port) {
		t.Fatalf("browser = %+v (skip, dashboard only)", rep.Browser)
	}

	want := "Server:    running on " + strconv.Itoa(port) + "\n" +
		"Autostart: enabled\n" +
		"PWA:       already installed -- dashboard: " + dashboardURLFor(port) + "\n"
	if got := renderSetupHuman(rep); got != want {
		t.Fatalf("render:\n%q\nwant:\n%q", got, want)
	}
}

func TestSetupPortChange(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldPort, newPort := 19712, 19713
	writeConfigWithPort(t, newPort)
	writePidFile(t, oldPort, os.Getpid())
	defer fakeServer(t, oldPort)()

	// Seed a real autostart entry pointing at the old port, so Enable's
	// rewrite reports prev == oldPort ("rewritten for N (was M)").
	am, err := autostart.New(autostart.Options{Port: oldPort})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := am.Enable(); err != nil {
		t.Fatal(err)
	}

	var killed, startedOld bool
	rep, ok := runSetup(newPort, setupRunner{
		kill: func() error { killed = true; return nil },
		start: func(p int) (*exec.Cmd, error) {
			if p == oldPort {
				startedOld = true
			}
			return nil, nil
		},
		serverUp: func(p int) bool { return true },
		openPWA:  chromiumOpener,
	})
	if !ok {
		t.Fatal("port-change setup must not gate")
	}
	if !killed || startedOld {
		t.Fatalf("port change must kill the old server and never start on the old port (killed=%v startedOld=%v)", killed, startedOld)
	}
	assertSetupServer(t, rep, "restarted", newPort, true)
	if rep.Autostart.PrevPort != oldPort {
		t.Fatalf("autostart prev = %d, want %d", rep.Autostart.PrevPort, oldPort)
	}
	if rep.PWA.Installed {
		t.Fatal("port change is a new origin — must not be treated as installed")
	}

	want := "Server:    starting on " + strconv.Itoa(newPort) + " ... OK\n" +
		"Autostart: rewritten for " + strconv.Itoa(newPort) + " (was " + strconv.Itoa(oldPort) + ")\n" +
		"PWA:       re-install from " + setupURL(newPort) + " (opening Edge)\n"
	if got := renderSetupHuman(rep); got != want {
		t.Fatalf("render:\n%q\nwant:\n%q", got, want)
	}
}

func TestSetupGateDaemonNeverCameUp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19714
	writeConfigWithPort(t, port)

	rep, ok := runSetup(port, setupRunner{
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return false },
		openPWA:  chromiumOpener,
	})
	if ok {
		t.Fatal("daemon-never-came-up must gate (exit 1)")
	}
	assertSetupServer(t, rep, "failed", port, false)
	if !strings.Contains(rep.Server.FailText, "did not come up on port") {
		t.Fatalf("fail text = %s", rep.Server.FailText)
	}

	// Gate renders server + one fix line only — soft sections empty.
	got := renderSetupHuman(rep)
	if !strings.Contains(got, "Server:    failed -- "+rep.Server.FailText) || !strings.Contains(got, "Fix:") {
		t.Fatalf("gate render:\n%s", got)
	}
	if strings.Contains(got, "Autostart:") || strings.Contains(got, "PWA:") {
		t.Fatalf("gate render must omit soft sections:\n%s", got)
	}

	// JSON stays clean: soft sections omitted via omitempty.
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "autostart") || strings.Contains(string(data), "pwa") {
		t.Fatalf("gate JSON must omit soft sections:\n%s", data)
	}
	if !strings.Contains(string(data), "running") {
		t.Fatalf("gate JSON missing running=false:\n%s", data)
	}
}

func TestSetupGateBusyPort(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19715
	writeConfigWithPort(t, port)
	defer fakeServer(t, port)()

	rep, ok := runSetup(port, setupRunner{
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return false },
		openPWA:  chromiumOpener,
	})
	if ok {
		t.Fatal("busy port must gate")
	}
	if !strings.Contains(rep.Server.FailText, "in use by another process") {
		t.Fatalf("fail text = %s", rep.Server.FailText)
	}
}

// TestSetupGateDeadPidfilePid covers the edge case where a foreign process
// holds the port but the pidfile points at a dead daemon (the child wrote
// its placeholder before exiting). processAlive rejects the dead pid.
func TestSetupGateDeadPidfilePid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows processAlive is a no-op this cycle")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19720
	writeConfigWithPort(t, port)
	defer fakeServer(t, port)()
	// Dead pid (almost certainly not alive) to simulate a crashed child.
	writePidFile(t, port, 999999999)

	rep, ok := runSetup(port, setupRunner{
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return false },
		openPWA:  chromiumOpener,
	})
	if ok {
		t.Fatal("dead pid + foreign port must gate")
	}
	if !strings.Contains(rep.Server.FailText, "in use by another process") {
		t.Fatalf("fail text = %s", rep.Server.FailText)
	}
}

func TestSetupRestartFailureGates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldPort, newPort := 19716, 19717
	writeConfigWithPort(t, newPort)
	writePidFile(t, oldPort, os.Getpid())
	defer fakeServer(t, oldPort)()

	rep, ok := runSetup(newPort, setupRunner{
		kill:     func() error { return fmt.Errorf("stop failed") },
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return true },
		openPWA:  chromiumOpener,
	})
	if ok {
		t.Fatal("kill failure in the port-change path must gate")
	}
	if rep.Server.Action != "failed" || !strings.Contains(rep.Server.FailText, "could not stop") {
		t.Fatalf("server = %+v", rep.Server)
	}
}

func TestSetupAutostartFailureIsSoft(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19718
	writeConfigWithPort(t, port)

	// Park a directory at the entry path: the autostart writer hits EISDIR
	// and the step fails — but the server step already succeeded, so runSetup
	// must report the failure as a soft warning, not a gate.
	if err := os.MkdirAll(filepath.Join(configDirFor(t), "autostart", "pharos.desktop"), 0o755); err != nil {
		t.Fatal(err)
	}

	rep, ok := runSetup(port, setupRunner{
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return true },
		openPWA:  chromiumOpener,
	})
	if !ok {
		t.Fatal("autostart failure is soft — must not gate")
	}
	if rep.Autostart.Status != "failed" || rep.Autostart.Err == "" {
		t.Fatalf("autostart = %+v", rep.Autostart)
	}
	if got := renderSetupHuman(rep); !strings.Contains(got, "Autostart: failed --") {
		t.Fatalf("render:\n%s", got)
	}
}

func TestSetupJSONKeyOrderAndShape(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	port := 19719
	writeConfigWithPort(t, port)

	rep, ok := runSetup(port, setupRunner{
		start:    func(p int) (*exec.Cmd, error) { return nil, nil },
		serverUp: func(p int) bool { return true },
		openPWA:  chromiumOpener,
	})
	if !ok {
		t.Fatal("must not gate")
	}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Flat struct, fixed key order: server, autostart, pwa, browser.
	order := []string{`"server"`, `"autostart"`, `"pwa"`, `"browser"`}
	last := -1
	for _, key := range order {
		idx := strings.Index(s, key)
		if idx < 0 {
			t.Fatalf("key %s missing in %s", key, s)
		}
		if idx < last {
			t.Fatalf("key order violated: %s at %d after %d in %s", key, idx, last, s)
		}
		last = idx
	}
	// Console-only fields never leak into JSON.
	for _, leak := range []string{`"Action"`, `"FailText"`, `"PrevPort"`, `"Name"`, `"Err"`} {
		if strings.Contains(s, leak) {
			t.Fatalf("console-only field %s leaked into JSON: %s", leak, s)
		}
	}
	var decoded setupReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Server.Running || decoded.Server.Port != port || decoded.Browser.Method != "chromium" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func assertSetupServer(t *testing.T, rep setupReport, action string, port int, running bool) {
	t.Helper()
	if rep.Server.Action != action || rep.Server.Port != port || rep.Server.Running != running {
		t.Fatalf("server = %+v, want action=%s port=%d running=%v", rep.Server, action, port, running)
	}
	if rep.Server.URL != dashboardURLFor(port) {
		t.Fatalf("server url = %s", rep.Server.URL)
	}
}
