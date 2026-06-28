package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quizzes in a workspace",
	Long: `List all quizzes for a workspace.

Examples:
  pharos quiz list --workspace "sql-for-research"
  pharos quiz list --workspace "sql-for-research" --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, listSpec[db.Quiz]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.Quiz, error) {
				return ws.GetQuizzes()
			},
			errLabel:   "failed to list quizzes",
			emptyMsg:   "No quizzes yet.",
			createHint: `pharos quiz create "Title" --workspace %q --items "slug1,slug2"`,
			headers:    []string{"Slug", "Title", "Items"},
			buildRow: func(q db.Quiz) []string {
				items, _ := q.ParseItems()
				return []string{
					q.Slug,
					truncate(q.Title, 40),
					fmt.Sprintf("%d", len(items)),
				}
			},
		})
	},
}

func init() {
	quizCmd.AddCommand(quizListCmd)
	quizListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
