//go:build !windows

package cli

import "syscall"

// processAlive probes whether a pid belongs to a live process:
// signal 0 performs an existence check without delivering anything.
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
