//go:build !windows

package cli

import "os/exec"

// On Unix the daemon already survives session lock. No detachment
// flags are needed — the child becomes orphaned and is reparented to
// init when the parent exits, and SIGHUP from a closing terminal is
// not a reported issue. Kept as a no-op for interface symmetry with
// daemon_windows.go.
func detachDaemon(c *exec.Cmd) {}
