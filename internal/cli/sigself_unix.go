//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// signalSelfTerminate sends SIGTERM to the current process — the graceful
// shutdown trigger server.Start listens for. No portable windows
// equivalent; the windows helper reports unsupported.
func signalSelfTerminate() error {
	return syscall.Kill(os.Getpid(), syscall.SIGTERM)
}
