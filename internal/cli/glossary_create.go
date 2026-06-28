package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var glossaryCreateCmd = &cobra.Command{
	Use:   "create <term> <definition>",
	Short: "Add or update a glossary term",
	Long: `Add a glossary term to a workspace.

If a term with the same name (case-insensitive) already exists, its
definition is updated. Pass --category to group terms under a heading
(e.g. "Diagnostic & Clinical") and --avoid to flag synonyms to avoid.

Examples:
  pharos glossary create "Hypertrophy" "Muscle growth from tension and stress"
  pharos glossary create "JOIN" "Combines rows from two tables on a condition" -w "sql-for-research"
  pharos glossary create "ASD" "Autism spectrum disorder diagnostic label" --category "Diagnostic" --avoid "Autism (without spectrum)"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		term := args[0]
		definition := args[1]
		category, _ := cmd.Flags().GetString("category")
		avoid, _ := cmd.Flags().GetString("avoid")
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		if err := wsStore.AddGlossaryTerm(term, definition, category, avoid); err != nil {
			return formatError("failed to add glossary term", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Glossary created: %s\n", term)
		fmt.Println()
		return nil
	},
}

func init() {
	glossaryCmd.AddCommand(glossaryCreateCmd)
	glossaryCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	glossaryCreateCmd.Flags().String("category", "", "Heading to group the term under (e.g. \"Diagnostic & Clinical\")")
	glossaryCreateCmd.Flags().String("avoid", "", "Synonyms or phrasing to avoid for this term")
}
