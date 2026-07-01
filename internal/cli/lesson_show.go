package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lessonShowCmd = &cobra.Command{
	Use:   "show <seq>",
	Short: "Get a lesson's dashboard URL",
	Long: `Print the dashboard URL for viewing a lesson.

Examples:
  pharos lesson show 3
  pharos lesson show 3 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := parseSeq(args[0])
		if err != nil {
			return err
		}
		return runShow(cmd, showSpec{
			urlPath: func(wsName string) string {
				return fmt.Sprintf("/w/%s/lessons/%d", wsName, n)
			},
			label: "lesson",
		})
	},
}

func init() {
	lessonCmd.AddCommand(lessonShowCmd)
	lessonShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
