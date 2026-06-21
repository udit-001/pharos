package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Show or update the workspace mission",
	Long: `Display or edit the MISSION.md file for a workspace.

The mission captures why you're learning a topic and what
success looks like. Every lesson should trace back to it.

Examples:
  learn mission --workspace "sql-for-research"
  learn mission --workspace "yoga" --edit`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		missionPath := filepath.Join(ws.Path, "MISSION.md")

		edit, _ := cmd.Flags().GetBool("edit")
		if edit {
			// Open MISSION.md in $EDITOR
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vim"
			}

			fmt.Println()
			fmt.Printf("  Opening MISSION.md in %s ...\n", editor)
			fmt.Println()

			// Launch editor with stdin/stdout connected
			editorCmd := execCommand(editor, missionPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			_ = wsStore.Touch()
			fmt.Println()
			fmt.Printf("  ✓ Mission updated\n")
			fmt.Println()
			return nil
		}

		// Read and print mission
		data, err := os.ReadFile(missionPath)
		if err != nil {
			return fmt.Errorf("read MISSION.md: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.Name)
		fmt.Println()
		fmt.Println(string(data))
		fmt.Println()

		// Also update DB
		_ = wsStore.Touch()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(missionCmd)
	missionCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	missionCmd.Flags().BoolP("edit", "e", false, "Open in $EDITOR")
}
