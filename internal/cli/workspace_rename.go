package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename a workspace (update its display name)",
	Long: `Rename a workspace by updating its display name (topic).

The directory slug and workspace key stay unchanged — only the
human-friendly display name is updated.

Examples:
  pharos workspace rename "Advanced SQL"
  pharos workspace rename "Yoga for Beginners" -w "yoga"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		newName := args[0]

		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		oldName := ws.DisplayName()

		if err := wsStore.UpdateTopic(newName); err != nil {
			return formatError("failed to rename workspace", err)
		}

		if jsonOut {
			printJSON(map[string]any{
				"renamed":  true,
				"old_name": oldName,
				"new_name": newName,
				"slug":     ws.Name,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Renamed workspace: %s\n", oldName)
		fmt.Printf("    New name: %s\n", newName)
		fmt.Printf("    Slug (unchanged): %s\n", ws.Name)
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceRenameCmd)
	workspaceRenameCmd.Flags().StringP("workspace", "w", "", "Workspace name (defaults to current)")
}
