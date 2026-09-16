//go:build windows

package cli

// processAlive on Windows: os.FindProcess is not a liveness probe and
// there is no portable signal-0 equivalent. Port health decides running
// state here; the unix builds gain the dead-pid guard. (The busy-port gate
// on Windows is therefore decided by pidfile-port + port health.)
func processAlive(pid int) bool {
	return true
}
