package cli

import (
	"fmt"

	"github.com/udit/learn-tool/internal/db"
	"github.com/spf13/cobra"
)

var refListCmd = &cobra.Command{
	Use:   "list",
	Short: "List reference documents in a workspace",
	Long: `List all reference documents for a workspace.

Examples:
  learn reference list --workspace "sql-for-research"
  learn reference list --workspace "sql-for-research" --search "join"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, listSpec[db.Reference]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.Reference, error) {
				if search != "" {
					return ws.SearchRefs(search)
				}
				return ws.GetRefs()
			},
			errLabel:   "failed to list references",
			emptyMsg:   "No references yet.",
			createHint: `learn reference create "Title" --workspace %q`,
			headers:    []string{"#", "Title", "File"},
			buildRow: func(r db.Reference) []string {
				return []string{
					fmt.Sprintf("%d", r.SequenceNumber),
					truncate(r.Title, 40),
					r.Filename,
				}
			},
		})
	},
}

func init() {
	refCmd.AddCommand(refListCmd)
	refListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	refListCmd.Flags().String("search", "", "Search references by title or summary")
}
