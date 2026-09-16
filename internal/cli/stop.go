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
	stopAlreadyStopped                    // pid file names a process that is already dead
	stopStopped                           // graceful stop delivered
)

// stopServerByPidfile stops the server named in the pid file and removes it —
// the shared routine behind `pharos stop` and `pharos setup`'s port-change
// path (LEARN-166 #281: stopProcess + cleanupPidFile = same path as stop).
// Missing / stale / dead-pid pidfiles each resolve to a distinct no-op
// outcome instead of an error, matching `pharos stop`'s script-friendly
// semantics. Callers tolerate every outcome; the outcome lets a caller
// distinguish "stopped" from "there was nothing to stop".
func stopServerByPidfile() stopOutcome {
	info, err := readPidFile()
	if err != nil {
		return stopNoServer
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
