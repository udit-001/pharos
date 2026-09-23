//go:build windows

package cli

// stopIdentityCheck gates `pharos stop` on process identity before any
// signal is sent (LEARN-235: a stale or reused pidfile must not kill an
// unrelated process).
//
// On Windows there is no portable liveness or identity probe: os.FindProcess
// succeeds for any pid, processAlive is a no-op, and stopProcess runs
// `taskkill /PID <pid> /T /F` — a force-kill of the whole process tree.
// A reused pid therefore gets its tree destroyed on pidfile evidence alone.
// The gate is port health instead: only taskkill when a server is actually
// listening on the pidfile's port. No healthy listener → nothing to stop,
// no kill, stale pidfile cleaned up.

func stopIdentityCheck(info *pidInfo) (bool, stopOutcome) {
	if !isServerRunning(info.Port) {
		return false, stopAlreadyStopped
	}
	return true, stopNoServer
}
