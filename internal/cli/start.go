package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/udit-001/pharos/internal/server"
	"github.com/spf13/cobra"
)

var startFlags struct {
	port       int
	noOpen     bool
	foreground bool
	background bool
	daemon     bool
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the web UI dashboard",
	Long: `Start a local web server with a read-only learning dashboard.

Shows your workspaces, lesson progress, and stats — all read-only.
Use the CLI commands to create lessons and learning records.

Examples:
  pharos start
  pharos start --port 9090
  pharos start --foreground`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		background := startFlags.background && !startFlags.foreground
		if background && !startFlags.daemon {
			args := []string{
				os.Args[0], "start",
				"--port", strconv.Itoa(startFlags.port),
				"--no-open",
				"--daemon",
			}
			c := exec.Command(args[0], args[1:]...)
			c.Stdin = nil
			c.Stdout = nil
			c.Stderr = nil
			if err := c.Start(); err != nil {
				return fmt.Errorf("failed to start background server: %w", err)
			}
			fmt.Println()
			fmt.Printf("  Pharos server started in background (PID: %d)\n", c.Process.Pid)
			fmt.Printf("  http://127.0.0.1:%d\n", startFlags.port)
			fmt.Println()
			return nil
		}

		if !startFlags.daemon {
			fmt.Println()
			fmt.Println("  Starting Pharos dashboard...")
			fmt.Println()
		}

		return server.Start(server.Config{
			Port:   startFlags.port,
			DB:     s,
			NoOpen: startFlags.noOpen,
			Silent: startFlags.daemon,
		})
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVar(&startFlags.port, "port", 9090, "HTTP server port (auto-increments if busy)")
	startCmd.Flags().BoolVar(&startFlags.noOpen, "no-open", false, "Don't auto-open browser")
	startCmd.Flags().BoolVarP(&startFlags.foreground, "foreground", "f", false, "Run server in foreground")
	startCmd.Flags().BoolVarP(&startFlags.background, "background", "b", true, "Run server in background")
	startCmd.Flags().BoolVar(&startFlags.daemon, "daemon", false, "")
	startCmd.Flags().MarkHidden("daemon")
}
