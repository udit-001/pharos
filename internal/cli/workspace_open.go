package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var workspaceOpenCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open a workspace directory",
	Long: `Open a workspace's directory in the file manager, or print its path.

Examples:
  learn workspace open "sql-for-research"
  learn workspace open "jump-start-a-car"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]

		wsStore, err := s.Workspace(name)
		if err != nil {
			return fmt.Errorf("workspace %q not found\n  Use 'learn workspace list' to see available workspaces", name)
		}
		ws := wsStore.Workspace()

		_ = wsStore.Touch()

		if jsonOut {
			printJSON(ws)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.Name)
		fmt.Printf("  Path: %s\n", ws.Path)
		fmt.Printf("  Lessons: %d  |  Learning Records: %d\n\n", ws.LessonCount, ws.RecordCount)

		// Open in file manager if supported
		openDir, _ := cmd.Flags().GetBool("open")
		if openDir {
			fmt.Printf("  Opening %s ...\n", ws.Path)
			openDirInExplorer(ws.Path)
		} else {
			fmt.Printf("  Use --open to open in file manager, or 'cd %s'\n", ws.Path)
		}
		fmt.Println()
		return nil
	},
}

func openDirInExplorer(path string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "explorer"
		args = []string{path}
	case "darwin":
		cmd = "open"
		args = []string{path}
	default:
		cmd = "xdg-open"
		args = []string{path}
	}
	exec.Command(cmd, args...).Start()
}

func init() {
	workspaceCmd.AddCommand(workspaceOpenCmd)
	workspaceOpenCmd.Flags().Bool("open", false, "Open in file manager")
}
