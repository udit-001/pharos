package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var documentQueryCmd = &cobra.Command{
	Use:   "query <terms...>",
	Short: "Retrieve passages from ingested source documents",
	Long: `Retrieve provenanced passages from the workspace's source documents.
Results carry {source id, title, chapter, excerpt} so a lesson can cite the
grounding. This is the dedicated source-retrieval surface — source text never
appears in 'pharos search', which indexes authored content only.

Grounding rule: every claim you write must map to a returned excerpt. If a
query returns nothing, REPHRASE the terms and query again — never invent.

Examples:
  pharos document query "mitochondria atp" --workspace "bio"
  pharos document query "join order" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		query := strings.Join(args, " ")
		hits, err := wsStore.QuerySources(query)
		if err != nil {
			return formatError("query sources", err)
		}

		if jsonOut {
			printJSON(hits)
			return nil
		}

		if len(hits) == 0 {
			fmt.Println()
			fmt.Println("  No matches in source documents. Rephrase the terms and query again.")
			fmt.Println()
			return nil
		}
		fmt.Println()
		for i, h := range hits {
			fmt.Printf("  [%d] %s (source %d)\n", i+1, h.Title, h.SourceID)
			if h.Chapter != "" {
				fmt.Printf("      %s\n", h.Chapter)
			}
			fmt.Printf("      %s\n", oneLine(h.Excerpt))
		}
		fmt.Println()
		return nil
	},
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func init() {
	documentCmd.AddCommand(documentQueryCmd)
	documentQueryCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
