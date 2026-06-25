package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var glossaryAddCmd = &cobra.Command{
	Use:   "add <term> <definition>",
	Short: "Add or update a glossary term",
	Long: `Add a glossary term to a workspace.

If a term with the same name (case-insensitive) already exists, its
definition is updated. Pass --category to group terms under a heading
(e.g. "Diagnostic & Clinical") and --avoid to flag synonyms to avoid.

Examples:
  pharos glossary add "Hypertrophy" "Muscle growth from tension and stress"
  pharos glossary add "JOIN" "Combines rows from two tables on a condition" -w "sql-for-research"
  pharos glossary add "ASD" "Autism spectrum disorder diagnostic label" --category "Diagnostic" --avoid "Autism (without spectrum)"`,
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

		fmt.Printf("Added %q to glossary.\n", term)
		return nil
	},
}

func init() {
	glossaryCmd.AddCommand(glossaryAddCmd)
	glossaryAddCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	glossaryAddCmd.Flags().String("category", "", "Heading to group the term under (e.g. \"Diagnostic & Clinical\")")
	glossaryAddCmd.Flags().String("avoid", "", "Synonyms or phrasing to avoid for this term")
}
