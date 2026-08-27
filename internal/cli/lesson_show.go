package cli

import (
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var lessonShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Get a lesson's dashboard URL",
	Long: `Print the dashboard URL for viewing a lesson.

Examples:
  pharos lesson show sql-joins
  pharos lesson show sql-joins --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		return runShow(cmd, showSpec{
			urlPath: func(wsName string) string {
				return urls.Lesson(wsName, slug)
			},
			label: "lesson",
		})
	},
}

func init() {
	lessonCmd.AddCommand(lessonShowCmd)
	lessonShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
