package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Show or edit the workspace resources",
	Long: `Display or edit the RESOURCES.md file for a workspace.

Resources are the curated set of trusted sources for this topic.
Knowledge comes from high-quality resources listed here.

Examples:
  pharos resources --workspace "sql-for-research"
  pharos resources --workspace "yoga" --edit`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		resourcesPath := filepath.Join(ws.Path, "RESOURCES.md")

		edit, _ := cmd.Flags().GetBool("edit")
		if edit {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}

			fmt.Println()
			fmt.Printf("  Opening RESOURCES.md in %s ...\n", editor)
			fmt.Println()

			editorCmd := execCommand(editor, resourcesPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			_ = wsStore.Touch()
			fmt.Println()
			fmt.Printf("  ✓ Resources updated\n")
			fmt.Println()
			return nil
		}

		data, err := os.ReadFile(resourcesPath)
		if err != nil {
			return fmt.Errorf("read RESOURCES.md: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.DisplayName())
		fmt.Println()
		fmt.Println(string(data))
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resourcesCmd)
	resourcesCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	resourcesCmd.Flags().BoolP("edit", "e", false, "Open in $EDITOR")
}
