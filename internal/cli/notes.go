package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage the workspace notes",
	Long: `Display or edit the NOTES.md file for a workspace.

Notes are a scratchpad for preferences and working notes. Use
--body-file to write non-interactively, or --append with --body-file
to add to the end instead of overwriting.

Examples:
  pharos notes read --workspace "sql-for-research"
  pharos notes edit --body-file /tmp/notes.md
  pharos notes edit --append --body-file /tmp/new-note.md
  pharos notes edit`,
}

var notesReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the workspace notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		notesPath := wsStore.Layout().NotesPath()
		if jsonOut {
			return readAndPrintJSON(ws, notesPath, "NOTES.md")
		}
		return readAndPrintFile(ws, notesPath)
	},
}

var notesEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the workspace notes",
	Long: `Update the NOTES.md file from a file, append to it, or open in $EDITOR.

Examples:
  pharos notes edit --body-file /tmp/notes.md
  pharos notes edit --append --body-file /tmp/new-note.md
  pharos notes edit`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		notesPath := wsStore.Layout().NotesPath()

		bodyFile, _ := cmd.Flags().GetString("body-file")
		appendMode, _ := cmd.Flags().GetBool("append")

		if bodyFile != "" {
			data, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			if appendMode {
				existing, _ := os.ReadFile(notesPath)
				content := string(existing) + string(data)
				if err := os.WriteFile(notesPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("append to notes: %w", err)
				}
			} else {
				if err := os.WriteFile(notesPath, data, 0o644); err != nil {
					return fmt.Errorf("write notes: %w", err)
				}
			}
			_ = wsStore.Touch()
			fmt.Println()
			fmt.Printf("  ✓ Notes updated\n")
			fmt.Println()
			return nil
		}

		return openInEditor(wsStore, notesPath, "NOTES.md")
	},
}

func init() {
	rootCmd.AddCommand(notesCmd)
	notesCmd.AddCommand(notesReadCmd)
	notesCmd.AddCommand(notesEditCmd)
	notesReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	notesEditCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	notesEditCmd.Flags().String("body-file", "", "Write content from a file (non-interactive)")
	notesEditCmd.Flags().Bool("append", false, "Append to notes instead of overwriting")
}
