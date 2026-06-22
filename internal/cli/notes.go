package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Show or edit the workspace notes",
	Long: `Display or edit the NOTES.md file for a workspace.

Notes are a scratchpad for preferences and working notes. Use
--body-file to write non-interactively, or --edit to open in $EDITOR.
Use --append with --body-file to add to the end instead of overwriting.

Examples:
  pharos notes --workspace "sql-for-research"
  pharos notes --workspace "yoga" --edit
  pharos notes --workspace "yoga" --body-file /tmp/notes.md
  pharos notes --workspace "yoga" --append --body-file /tmp/new-note.md`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		notesPath := wsStore.Layout().NotesPath()

		edit, _ := cmd.Flags().GetBool("edit")
		bodyFile, _ := cmd.Flags().GetString("body-file")
		append, _ := cmd.Flags().GetBool("append")

		if bodyFile != "" {
			data, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			if append {
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
			fmt.Println()
			fmt.Printf("  ✓ Notes updated\n")
			fmt.Println()
			return nil
		}

		if edit {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vim"
			}

			fmt.Println()
			fmt.Printf("  Opening NOTES.md in %s ...\n", editor)
			fmt.Println()

			editorCmd := execCommand(editor, notesPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			fmt.Println()
			fmt.Printf("  ✓ Notes updated\n")
			fmt.Println()
			return nil
		}

		// Read and print notes
		data, err := os.ReadFile(notesPath)
		if err != nil {
			return fmt.Errorf("read NOTES.md: %w", err)
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
	rootCmd.AddCommand(notesCmd)
	notesCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	notesCmd.Flags().BoolP("edit", "e", false, "Open in $EDITOR")
	notesCmd.Flags().String("body-file", "", "Write content from a file (non-interactive)")
	notesCmd.Flags().Bool("append", false, "Append to notes instead of overwriting")
}
