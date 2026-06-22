package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the current workspace",
	Long: `Set the current workspace so subsequent commands default to it
without needing --workspace.

Examples:
  pharos workspace use "sql-for-research"
  pharos workspace use "yoga"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]

		wsStore, err := s.Workspace(name)
		if err != nil {
			return fmt.Errorf("workspace %q not found\n  Use 'pharos workspace list' to see available workspaces", name)
		}
		ws := wsStore.Workspace()

		if err := s.SetCurrentWorkspace(name); err != nil {
			return formatError("failed to set current workspace", err)
		}

		if jsonOut {
			printJSON(ws)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Current workspace: %s\n", ws.DisplayName())
		fmt.Printf("    %d lessons · %d records · %d refs\n", ws.LessonCount, ws.RecordCount, ws.RefCount)
		fmt.Println()
		return nil
	},
}

var workspaceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current workspace",
	Long:  `Print the name of the current workspace, or nothing if none is set.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name, err := s.CurrentWorkspace()
		if err != nil {
			return formatError("failed to get current workspace", err)
		}

		if name == "" {
			if jsonOut {
				printJSON(nil)
				return nil
			}
			fmt.Println()
			fmt.Println("  No current workspace set.")
			fmt.Println("  Use 'pharos workspace use <name>' to set one.")
			fmt.Println()
			return nil
		}

		wsStore, err := s.Workspace(name)
		if err != nil {
			// Workspace was deleted — clear the stale reference
			_ = s.SetCurrentWorkspace("")
			if jsonOut {
				printJSON(nil)
				return nil
			}
			fmt.Println()
			fmt.Println("  No current workspace set.")
			fmt.Println("  Use 'pharos workspace use <name>' to set one.")
			fmt.Println()
			return nil
		}
		ws := wsStore.Workspace()

		if jsonOut {
			printJSON(ws)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Current workspace: %s\n", ws.DisplayName())
		fmt.Printf("    %d lessons · %d records · %d refs\n", ws.LessonCount, ws.RecordCount, ws.RefCount)
		fmt.Println()
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceUseCmd)
	workspaceCmd.AddCommand(workspaceCurrentCmd)
}
