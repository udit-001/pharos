//go:build windows

package cli

import (
	"os/exec"
	"syscall"
)

// DETACHED_PROCESS (0x00000008) detaches the child from the parent's
// console. Without it, the daemon inherits the parent's console and
// receives CTRL_CLOSE_EVENT when the console is destroyed — which
// happens when the user locks the screen or disconnects an RDP session.
// The default console control handler calls ExitProcess, killing the
// daemon. This matches the pattern used by Go's own telemetry daemon
// (golang.org/x/telemetry/start_windows.go).
//
// See: https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
const detachedProcess = 0x00000008

func detachDaemon(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess,
	}
}
