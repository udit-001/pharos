package cli

import (
	"fmt"
	"os"
	"syscall"

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
		info, err := readPidFile()
		if err != nil {
			if jsonOut {
				printJSON(map[string]any{"running": false, "message": "no server running"})
				return nil
			}
			fmt.Println()
			fmt.Println("  No running Pharos server found")
			fmt.Println()
			return nil
		}

		proc, err := os.FindProcess(info.PID)
		if err != nil {
			// Process doesn't exist; clean up stale PID file
			cleanupPidFile()
			if jsonOut {
				printJSON(map[string]any{"running": false, "message": "stale PID file cleaned up"})
				return nil
			}
			fmt.Println()
			fmt.Println("  No running Pharos server found (stale PID file cleaned up)")
			fmt.Println()
			return nil
		}

		if err := proc.Signal(syscall.SIGINT); err != nil {
			// Process already dead; clean up stale PID file
			cleanupPidFile()
			if jsonOut {
				printJSON(map[string]any{"running": false, "message": "server already stopped"})
				return nil
			}
			fmt.Println()
			fmt.Println("  Pharos server already stopped (stale PID file cleaned up)")
			fmt.Println()
			return nil
		}

		cleanupPidFile()

		if jsonOut {
			printJSON(map[string]any{"running": false, "message": "server stopped"})
			return nil
		}
		fmt.Println()
		fmt.Println("  Pharos server stopped")
		fmt.Println()
		return nil
	},
}

func cleanupPidFile() {
	os.Remove(config.PidPath())
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
