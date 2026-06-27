package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a workspace and its directory",
	Long: `Delete a workspace from the database and remove its directory on disk.

Removes the workspace row (cascading to its lessons, records, and
references) and deletes the workspace directory. The deletion is
irreversible. Prompts for confirmation unless --force is given.

Examples:
  pharos workspace delete "sql-for-research"
  pharos workspace delete "jump-start-a-car" --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]

		wsStore, err := s.Workspace(name)
		if err != nil {
			return fmt.Errorf("workspace %q not found\n  Use 'pharos workspace list' to see available workspaces", name)
		}
		ws := wsStore.Workspace()

		force, _ := cmd.Flags().GetBool("force")

		if !force && !jsonOut {
			fmt.Println()
			fmt.Printf("  Delete workspace %q and all its files?\n", ws.DisplayName())
			fmt.Printf("  Path: %s\n", ws.Path)
			fmt.Printf("  Contents: %d lessons, %d records, %d references\n", ws.LessonCount, ws.RecordCount, ws.RefCount)
			fmt.Println()
			fmt.Print("  This cannot be undone. Continue? [y/N] ")

			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("  Cancelled.")
				return nil
			}
		}

		// DeleteWorkspaceByName owns the full teardown: DB row (cascading to
		// lessons/records/references), the on-disk directory, and clearing the
		// current-workspace setting if it pointed here. Confirmation is a UI
		// concern, handled above.
		if err := s.DeleteWorkspaceByName(ws.Name); err != nil {
			return err
		}

		if jsonOut {
			printJSON(map[string]any{
				"deleted": true,
				"name":    ws.Name,
				"path":    ws.Path,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Deleted workspace: %s\n", ws.DisplayName())
		fmt.Printf("    Path: %s\n", ws.Path)
		fmt.Printf("    Removed: %d lessons, %d records, %d references\n", ws.LessonCount, ws.RecordCount, ws.RefCount)
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceDeleteCmd)
	workspaceDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
