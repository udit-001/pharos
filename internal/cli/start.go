package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/server"
)

const defaultPort = 9090

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

If a server is already running, prints its URL and returns.
Otherwise, starts the server in the background and prints the URL.

The dashboard opens on the current workspace if one is set,
otherwise on the main dashboard page.

Examples:
  pharos start
  pharos start --port 9090
  pharos start --foreground`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		// Check if server is already running
		if info, err := readPidFile(); err == nil && isServerRunning(info.Port) {
			url := dashboardURL(s, info.Port)
			if jsonOut {
				printJSON(map[string]any{"url": url, "running": true, "port": info.Port})
				return nil
			}
			fmt.Println()
			fmt.Printf("  Pharos dashboard already running\n")
			fmt.Printf("  %s\n", url)
			fmt.Println()
			return nil
		}

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

			// Wait briefly for server to come up
			time.Sleep(500 * time.Millisecond)

			// Find the actual port from the PID file
			port := startFlags.port
			if info, err := readPidFile(); err == nil {
				port = info.Port
			}

			url := dashboardURL(s, port)
			if jsonOut {
				printJSON(map[string]any{"url": url, "running": true, "port": port})
				return nil
			}
			fmt.Println()
			fmt.Printf("  Pharos server started in background (PID: %d)\n", c.Process.Pid)
			fmt.Printf("  %s\n", url)
			fmt.Println()
			return nil
		}

		if !startFlags.daemon {
			fmt.Println()
			fmt.Println("  Starting Pharos dashboard...")
			fmt.Println()
		}

		// Write PID file before starting server
		home, _ := os.UserHomeDir()
		pidPath := filepath.Join(home, ".pharos", "server.pid")
		_ = os.MkdirAll(filepath.Dir(pidPath), 0o755)

		// The server will write the actual port to the PID file
		// after it binds. For now, write a placeholder.
		_ = os.WriteFile(pidPath, []byte(fmt.Sprintf(`{"port":%d,"pid":%d}`, startFlags.port, os.Getpid())), 0o644)

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
	startCmd.Flags().IntVar(&startFlags.port, "port", defaultPort, "HTTP server port (auto-increments if busy)")
	startCmd.Flags().BoolVar(&startFlags.noOpen, "no-open", false, "Don't auto-open browser")
	startCmd.Flags().BoolVarP(&startFlags.foreground, "foreground", "f", false, "Run server in foreground")
	startCmd.Flags().BoolVarP(&startFlags.background, "background", "b", true, "Run server in background")
	startCmd.Flags().BoolVar(&startFlags.daemon, "daemon", false, "")
	startCmd.Flags().MarkHidden("daemon")
}

// pidInfo represents the server.pid file content.
type pidInfo struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

func readPidFile() (*pidInfo, error) {
	home, _ := os.UserHomeDir()
	pidPath := filepath.Join(home, ".pharos", "server.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return nil, err
	}
	var info pidInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// isServerRunning checks if a server is actually listening on the given port.
func isServerRunning(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// dashboardURL returns the URL for the dashboard, preferring the current workspace.
func dashboardURL(s *db.Store, port int) string {
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	current, err := s.CurrentWorkspace()
	if err != nil || current == "" {
		return base + "/"
	}
	// Verify the workspace exists
	_, err = s.Workspace(current)
	if err != nil {
		return base + "/"
	}
	return fmt.Sprintf("%s/workspace/%s", base, strings.ReplaceAll(current, " ", "%20"))
}
