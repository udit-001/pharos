package cli

import (
	"fmt"

	"github.com/udit-001/pharos/internal/db"
	"github.com/spf13/cobra"
)

var lessonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List lessons in a workspace",
	Long: `List all lessons for a workspace, ordered by sequence number.

Examples:
  pharos lesson list --workspace "sql-for-research"
  pharos lesson list
  pharos lesson list --search "join"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd, listSpec[db.Lesson]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.Lesson, error) {
				if search != "" {
					return ws.SearchLessons(search)
				}
				return ws.GetLessons()
			},
			errLabel:   "failed to list lessons",
			emptyMsg:   "No lessons yet.",
			createHint: `pharos lesson create "Title" --workspace %q`,
			headers:    []string{"#", "Title", "File"},
			buildRow: func(l db.Lesson) []string {
				return []string{
					fmt.Sprintf("%d", l.SequenceNumber),
					truncate(l.Title, 40),
					l.Filename,
				}
			},
		})
	},
}

func init() {
	lessonCmd.AddCommand(lessonListCmd)
	lessonListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonListCmd.Flags().String("search", "", "Search lessons by title or summary")
}
