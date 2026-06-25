package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var glossaryCmd = &cobra.Command{
	Use:   "glossary",
	Short: "Display the workspace glossary",
	Long: `Display the glossary for a workspace.

The glossary records canonical terminology for this workspace.

Examples:
  pharos glossary
  pharos glossary --workspace "jump-start-a-car"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		terms, err := wsStore.GetGlossaryTerms()
		if err != nil {
			return fmt.Errorf("read glossary: %w", err)
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		if len(terms) == 0 {
			fmt.Println("  (no glossary terms yet)")
			fmt.Println()
			fmt.Println("  Add one with: pharos glossary add \"<term>\" \"<definition>\"")
			fmt.Println()
			return nil
		}

		prevCat := "__init__"
		for _, t := range terms {
			if t.Category != prevCat {
				if t.Category == "" {
					fmt.Println("  Other")
				} else {
					fmt.Printf("  %s\n", t.Category)
				}
				prevCat = t.Category
			}
			fmt.Printf("    %s\n", t.Term)
			fmt.Printf("      %s\n", t.Definition)
			if t.Avoid != "" {
				fmt.Printf("      avoid: %s\n", t.Avoid)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(glossaryCmd)
	glossaryCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
