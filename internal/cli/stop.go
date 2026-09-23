package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/config"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running web UI dashboard server",
	Long: `Stop the local Pharos web server if one is running.

Reads the server PID file, sends a graceful
shutdown signal (SIGINT) to the server process, and cleans up
the PID file.

Examples:
  pharos stop`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch stopServerByPidfile() {
		case stopNoServer:
			if jsonEnabled(cmd) {
				printJSON(map[string]any{"running": false, "message": "no server running"})
				return nil
			}
			fmt.Println()
			fmt.Println("  No running Pharos server found")
			fmt.Println()
		case stopStalePID:
			if jsonEnabled(cmd) {
				printJSON(map[string]any{"running": false, "message": "stale PID file cleaned up"})
				return nil
			}
			fmt.Println()
			fmt.Println("  No running Pharos server found (stale PID file cleaned up)")
			fmt.Println()
		case stopForeignPID:
			if jsonEnabled(cmd) {
				printJSON(map[string]any{"running": false, "message": "PID file named a non-Pharos process; removed PID file without signaling"})
				return nil
			}
			fmt.Println()
			fmt.Println("  No running Pharos server found")
			fmt.Println("  (PID file named a process that is not Pharos — removed it without signaling)")
			fmt.Println()
		case stopAlreadyStopped:
			if jsonEnabled(cmd) {
				printJSON(map[string]any{"running": false, "message": "server already stopped"})
				return nil
			}
			fmt.Println()
			fmt.Println("  Pharos server already stopped (stale PID file cleaned up)")
			fmt.Println()
		case stopStopped:
			if jsonEnabled(cmd) {
				printJSON(map[string]any{"running": false, "message": "server stopped"})
				return nil
			}
			fmt.Println()
			fmt.Println("  Pharos server stopped")
			fmt.Println()
		}
		return nil
	},
}

func cleanupPidFile() {
	os.Remove(config.PidPath())
}

// stopOutcome describes what stopServerByPidfile found.
type stopOutcome int

const (
	stopNoServer       stopOutcome = iota // no pid file
	stopStalePID                          // pid file names a process that no longer exists
	stopForeignPID                        // pid file names a live process that is not pharos — not signaled (LEARN-235)
	stopAlreadyStopped                    // pid file names a process that is already dead
	stopStopped                           // graceful stop delivered
)

// stopServerByPidfile stops the server named in the pid file and removes it —
// the shared routine behind `pharos stop` and `pharos setup`'s port-change
// path (LEARN-166 #281: stopProcess + cleanupPidFile = same path as stop).
// Missing / stale / dead-pid / foreign-pid pidfiles each resolve to a
// distinct no-op outcome instead of an error, matching `pharos stop`'s
// script-friendly semantics. Callers tolerate every outcome; the outcome
// lets a caller distinguish "stopped" from "there was nothing to stop".
//
// LEARN-235: before signaling, the pidfile's pid is identity-checked — a
// stale or reused pidfile must never let stop signal (or, on Windows,
// taskkill /T /F) an unrelated process. The check is per-OS: unix probes
// liveness + executable name; windows requires the port-health gate.
func stopServerByPidfile() stopOutcome {
	info, err := readPidFile()
	if err != nil {
		return stopNoServer
	}
	if proceed, outcome := stopIdentityCheck(info); !proceed {
		cleanupPidFile()
		return outcome
	}
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		cleanupPidFile()
		return stopStalePID
	}
	if err := stopProcess(proc); err != nil {
		cleanupPidFile()
		return stopAlreadyStopped
	}
	cleanupPidFile()
	return stopStopped
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
