//go:build !windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// processName best-effort resolves the executable name of a pid:
// /proc/<pid>/exe (Linux), falling back to /proc/<pid>/comm, then
// `ps -p <pid> -o comm=` (macOS has no /proc). Empty string means the name
// could not be determined.
func processName(pid int) string {
	if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
		return filepath.Base(exe)
	}
	if comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		if name := strings.TrimSpace(string(comm)); name != "" {
			return name
		}
	}
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if line == "" {
		return ""
	}
	// ps comm on macOS can be a full path; on Linux it is the command name.
	return filepath.Base(line)
}

// processIsPharos reports whether the pid belongs to a pharos binary.
// An undeterminable name counts as NOT pharos: stop refuses to signal
// what it cannot verify (LEARN-235 — loud refusal over wrong-process kill).
func processIsPharos(pid int) bool {
	name := processName(pid)
	return name == "pharos" ||
		name == "pharos.exe" ||
		strings.HasPrefix(name, "pharos.") // pharos.test, dev builds
}

// stopIdentityCheck gates `pharos stop` on process identity before any
// signal is sent (LEARN-235: a stale or reused pidfile must not signal an
// arbitrary process).
//
// On unix this is a real identity probe: the pid must be alive (signal-0;
// os.FindProcess succeeds even for dead pids) and resolve to a pharos
// executable. Returns (proceed, outcome); outcome is meaningful only when
// proceed is false and selects stop.go's refusal message.
func stopIdentityCheck(info *pidInfo) (bool, stopOutcome) {
	if !processAlive(info.PID) {
		// Dead pid → stale pidfile, nothing to signal.
		return false, stopStalePID
	}
	if !processIsPharos(info.PID) {
		// Live but not pharos: PID reuse. Never signal an unverified
		// process — clean the pidfile and stop there.
		return false, stopForeignPID
	}
	return true, stopNoServer
}
