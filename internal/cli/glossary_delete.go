package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var glossaryDeleteCmd = &cobra.Command{
	Use:   "delete <term>",
	Short: "Remove a glossary term",
	Long: `Remove a glossary term from a workspace by name (case-insensitive).

Does nothing if the term does not exist — delete is idempotent.

Examples:
  pharos glossary delete "Hypertrophy"
  pharos glossary delete "JOIN" -w "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		term := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		if err := wsStore.DeleteGlossaryTerm(term); err != nil {
			return formatError("failed to delete glossary term", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Deleted glossary term: %s\n", term)
		fmt.Println()
		return nil
	},
}

func init() {
	glossaryCmd.AddCommand(glossaryDeleteCmd)
	glossaryDeleteCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
