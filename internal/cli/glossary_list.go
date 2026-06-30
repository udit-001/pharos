package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var glossaryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List glossary terms",
	Long: `Display all glossary terms for a workspace, grouped by category.

Examples:
  pharos glossary list
  pharos glossary list --workspace "jump-start-a-car"
  pharos glossary list --json`,
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

		if jsonOut {
			if terms == nil {
				terms = []db.GlossaryTerm{}
			}
			type termJSON struct {
				Term       string `json:"term"`
				Definition string `json:"definition"`
				Category   string `json:"category,omitempty"`
				Avoid      string `json:"avoid,omitempty"`
			}
			result := make([]termJSON, 0, len(terms))
			for _, t := range terms {
				result = append(result, termJSON{
					Term:       t.Term,
					Definition: t.Definition,
					Category:   t.Category,
					Avoid:      t.Avoid,
				})
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		if len(terms) == 0 {
			fmt.Printf("  No glossary terms yet.\n")
			fmt.Printf("  Use 'pharos glossary create \"<term>\" \"<definition>\"' to add one.\n")
			fmt.Println()
			return nil
		}

		prevCat := "__init__"
		for _, t := range terms {
			if t.Category != prevCat {
				if t.Category == "" {
					fmt.Println()
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
	glossaryCmd.AddCommand(glossaryListCmd)
	glossaryListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
