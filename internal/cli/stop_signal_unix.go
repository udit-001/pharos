//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// stopProcess sends a graceful shutdown signal to the server process.
// On Unix, SIGINT allows the server to shut down cleanly (close DB,
// release the port) before exiting.
func stopProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGINT)
}
