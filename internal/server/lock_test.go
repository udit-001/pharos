package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"

	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/db"
)

// writePidFile mirrors the daemon's server.pid shape (JSON {port, pid}).
func writePidFile(t *testing.T, port, pid int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.PidPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]int{"port": port, "pid": pid})
	if err := os.WriteFile(config.PidPath(), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func holdServerLock(t *testing.T) *flock.Flock {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ServerLockPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	l := flock.New(config.ServerLockPath())
	locked, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("precondition: server lock must be free")
	}
	return l
}

// TestAcquireServerLockRefusedWhileHeld: the second server must fail loudly
// naming the running instance instead of racing it (LEARN-235).
func TestAcquireServerLockRefusedWhileHeld(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writePidFile(t, 9090, 424242)
	held := holdServerLock(t)
	defer held.Unlock()

	_, err := acquireServerLock()
	if err == nil {
		t.Fatal("second server must refuse while the server lock is held")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Fatalf("err = %v, want already-running refusal", err)
	}
	if !strings.Contains(err.Error(), "424242") {
		t.Fatalf("refusal must name the running instance's PID, got: %v", err)
	}
}

// TestAcquireServerLockSucceedsWhenFree: the happy path — a fresh boot takes
// the lock; releasing it makes it available again.
func TestAcquireServerLockSucceedsWhenHeld(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	release, err := acquireServerLock()
	if err != nil {
		t.Fatalf("fresh boot must take the server lock: %v", err)
	}
	if _, err := acquireServerLock(); err == nil {
		t.Fatal("second acquisition while held must fail")
	}
	release()
	if _, err := acquireServerLock(); err != nil {
		t.Fatalf("lock must be free after release: %v", err)
	}
}

// TestStartServerLockLifetime pins the lock to the serve lifecycle: held
// before the listener accepts, released when Start returns (SIGINT shutdown).
func TestStartServerLockLifetime(t *testing.T) {
	if os.Getenv("PHAROS_TEST_SIGSELF") == "" {
		t.Skip("sends SIGINT to the test process; opt-in so `-run` scoping stays safe")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Reserve a port for the test, free it, and hand the number to Start.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	done := make(chan error, 1)
	go func() { done <- Start(Config{Port: port, DB: store, NoOpen: true, Silent: true}) }()

	// Wait until the server is serving.
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not come up")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// While serving, the lock is ours: a second start must be refused.
	l := flock.New(config.ServerLockPath())
	locked, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatal("server lock free while a server is serving — double-start race window")
	}

	// Graceful shutdown releases the lock.
	if err := signalSelfInterrupt(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("start did not shut down after SIGINT")
	}
	locked, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("server lock not released after shutdown")
	}
	_ = l.Unlock()
}

// TestStartNamesRunningInstanceWithoutPidFile: the refusal must degrade
// gracefully when the pidfile is unreadable — still a loud already-running
// failure, just without a PID to name.
func TestStartNamesRunningInstanceWithoutPidFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	held := holdServerLock(t)
	defer held.Unlock()

	_, err := acquireServerLock()
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), strconv.Itoa(os.Getpid())) {
		t.Fatal("refusal must not invent a pid")
	}
}
