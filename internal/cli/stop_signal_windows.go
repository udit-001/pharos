//go:build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
)

// stopProcess kills the server process on Windows. Go's os.Process.Signal
// only supports os.Kill on Windows — SIGINT cannot be delivered to another
// process. We use taskkill /T to kill the process and any children it may
// have spawned (e.g. the daemon started by `pharos start --background`).
func stopProcess(proc *os.Process) error {
	return exec.Command("taskkill", "/PID", fmt.Sprintf("%d", proc.Pid), "/T", "/F").Run()
}
