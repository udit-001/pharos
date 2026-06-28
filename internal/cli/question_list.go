package cli

import (
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List questions in a workspace",
	Long: `List all questions for a workspace.

Examples:
  pharos question list --workspace "sql-for-research"
  pharos question list --workspace "sql-for-research" --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, listSpec[db.Question]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.Question, error) {
				return ws.GetQuestions()
			},
			errLabel:   "failed to list questions",
			emptyMsg:   "No questions yet.",
			createHint: `pharos question create "Title" --workspace %q --mode choice --body-file <path>`,
			headers:    []string{"Slug", "Title", "Mode"},
			buildRow: func(q db.Question) []string {
				return []string{
					q.Slug,
					truncate(q.Title, 40),
					q.Mode,
				}
			},
		})
	},
}

func init() {
	questionCmd.AddCommand(questionListCmd)
	questionListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
