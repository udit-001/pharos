package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a workspace and its directory",
	Long: `Delete a workspace from the database and remove its directory.

Removes the workspace row (cascading to its lessons, records, and
references) and deletes the workspace directory on disk. The deletion
is irreversible — use --dry-run to preview what would be removed.

Examples:
  pharos workspace delete "sql-for-research"
  pharos workspace delete "jump-start-a-car" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]

		wsStore, err := s.Workspace(name)
		if err != nil {
			return fmt.Errorf("workspace %q not found\n  Use 'pharos workspace list' to see available workspaces", name)
		}
		ws := wsStore.Workspace()

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if dryRun {
			if jsonOut {
				printJSON(map[string]any{
					"dry_run":  true,
					"name":     ws.Name,
					"path":     ws.Path,
					"lessons":  ws.LessonCount,
					"records":  ws.RecordCount,
					"refs":     ws.RefCount,
				})
				return nil
			}
			fmt.Println()
			fmt.Printf("  Would delete workspace: %s\n", ws.DisplayName())
			fmt.Printf("    Path: %s\n", ws.Path)
			fmt.Printf("    Lessons: %d  |  Records: %d  |  References: %d\n", ws.LessonCount, ws.RecordCount, ws.RefCount)
			fmt.Println()
			return nil
		}

		// Delete the DB row first (cascades to lessons, records, references)
		if err := s.DeleteWorkspace(ws.ID); err != nil {
			return fmt.Errorf("delete workspace from database: %w", err)
		}

		// Remove the workspace directory
		if err := os.RemoveAll(ws.Path); err != nil {
			return fmt.Errorf("remove workspace directory: %w", err)
		}

		if jsonOut {
			printJSON(map[string]any{
				"deleted":  true,
				"name":     ws.Name,
				"path":     ws.Path,
				"lessons":  ws.LessonCount,
				"records":  ws.RecordCount,
				"refs":     ws.RefCount,
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
	workspaceDeleteCmd.Flags().Bool("dry-run", false, "Preview what would be deleted without removing anything")
}