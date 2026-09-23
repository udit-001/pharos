//go:build !windows

package server

import (
	"os"
	"syscall"
)

// signalSelfInterrupt sends SIGINT to the current process — the graceful
// shutdown trigger server.Start listens for. Used by the serve-lifetime
// lock test; no portable equivalent exists on windows, where the helper
// reports unsupported instead.
func signalSelfInterrupt() error {
	return syscall.Kill(os.Getpid(), syscall.SIGINT)
}
