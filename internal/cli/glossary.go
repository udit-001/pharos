package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var glossaryCmd = &cobra.Command{
	Use:   "glossary",
	Short: "Show or edit the workspace glossary",
	Long: `Display or edit the GLOSSARY.md file for a workspace.

The glossary is the canonical language for this teaching workspace.
It records terminology with opinionated definitions.

Examples:
  learn glossary --workspace "jump-start-a-car"
  learn glossary --workspace "cell-adhesion" --edit`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		glossaryPath := filepath.Join(ws.Path, "GLOSSARY.md")

		edit, _ := cmd.Flags().GetBool("edit")
		if edit {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}

			fmt.Println()
			fmt.Printf("  Opening GLOSSARY.md in %s ...\n", editor)
			fmt.Println()

			editorCmd := execCommand(editor, glossaryPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			wsStore.Touch()
			fmt.Println()
			fmt.Printf("  ✓ Glossary updated\n")
			fmt.Println()
			return nil
		}

		data, err := os.ReadFile(glossaryPath)
		if err != nil {
			return fmt.Errorf("read GLOSSARY.md: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.Name)
		fmt.Println()
		fmt.Println(string(data))
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(glossaryCmd)
	glossaryCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	glossaryCmd.Flags().BoolP("edit", "e", false, "Open in $EDITOR")
}
