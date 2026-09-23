package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"

	"github.com/udit-001/pharos/internal/config"
)

// acquireServerLock takes the exclusive non-blocking lock on server.lock —
// the singleton guard for the dashboard server (LEARN-235). A held lock
// means another Pharos server is already serving: it fails with a
// plain-language message naming the running instance instead of racing it.
// The returned release func must be called when the server stops.
//
// This closes the double-start TOCTOU that the pidfile + HTTP health check
// cannot: two rapid `pharos start` invocations can both pass the health
// gate before either daemon binds, but only one can hold this lock.
func acquireServerLock() (release func(), err error) {
	if err := os.MkdirAll(filepath.Dir(config.ServerLockPath()), 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	l := flock.New(config.ServerLockPath())
	locked, err := l.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire server lock %s: %w", config.ServerLockPath(), err)
	}
	if !locked {
		if info, perr := config.ReadPidFile(); perr == nil {
			return nil, fmt.Errorf(
				"another Pharos server is already running (PID %d, port %d) — the dashboard is at http://127.0.0.1:%d\n\n  Fix: nothing to start. Open the URL, or run 'pharos stop' to shut the running server down",
				info.PID, info.Port, info.Port)
		}
		return nil, errors.New(
			"another Pharos server is already running\n\n  Fix: run 'pharos stop' to shut it down before starting another")
	}
	return func() { _ = l.Unlock() }, nil
}
