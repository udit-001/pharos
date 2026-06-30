package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var assetDeleteCmd = &cobra.Command{
	Use:   "delete <filename>",
	Short: "Remove an asset file",
	Long: `Remove a file from the workspace's assets/ directory. No prompt —
assets have no database cascade and recreate cheaply (re-add or re-create).

Filename-based, like 'create' — not name-based. To remove a whole vendored
asset set, delete each file; 'pharos asset redeploy' self-heals a partial
state if you only meant to refresh.

Examples:
  pharos asset delete quiz-widget.js
  pharos asset delete mermaid.min.js`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		filename := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		if err := wsStore.DeleteAsset(filename); err != nil {
			return fmt.Errorf("delete asset: %w", err)
		}
		_ = wsStore.Touch()

		assetPath := filepath.Join(ws.Path, "assets", filename)

		if jsonOut {
			type assetResult struct {
				Deleted   bool   `json:"deleted"`
				Filename  string `json:"filename"`
				Path      string `json:"path"`
				Workspace string `json:"workspace"`
			}
			printJSON(assetResult{
				Deleted:   true,
				Filename:  filename,
				Path:      assetPath,
				Workspace: ws.Name,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Asset deleted: %s\n", filename)
		fmt.Printf("    File: %s\n", assetPath)
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

func init() {
	assetCmd.AddCommand(assetDeleteCmd)
	assetDeleteCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
