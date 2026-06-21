package cli

import (
	"fmt"

	"github.com/udit-001/pharos/internal/db"
	"github.com/spf13/cobra"
)

var recordListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learning records in a workspace",
	Long: `List all learning records for a workspace, ordered by sequence number.

Examples:
  learn record list --workspace "sql-for-research"
  learn record list
  learn record list --search "join"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, listSpec[db.LearningRecord]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.LearningRecord, error) {
				if search != "" {
					return ws.SearchRecords(search)
				}
				return ws.GetRecords()
			},
			errLabel:   "failed to list learning records",
			emptyMsg:   "No learning records yet.",
			createHint: `learn record add "What you learned" --workspace %q`,
			headers:    []string{"#", "Title", "File"},
			buildRow: func(r db.LearningRecord) []string {
				status := ""
				if r.Status == "superseded" {
					status = " (superseded)"
				}
				return []string{
					fmt.Sprintf("%d", r.SequenceNumber),
					truncate(r.Title, 40) + status,
					r.Filename,
				}
			},
		})
	},
}

func init() {
	recordCmd.AddCommand(recordListCmd)
	recordListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	recordListCmd.Flags().String("search", "", "Search records by title or summary")
}
