package cli

import (
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var refShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Get a reference document's dashboard URL",
	Long: `Print the dashboard URL for viewing a reference document.

Examples:
  pharos reference show sql-syntax
  pharos reference show sql-syntax --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		return runShow(cmd, showSpec{
			urlPath: func(wsName string) string {
				return urls.Ref(wsName, slug)
			},
			label: "reference",
		})
	},
}

func init() {
	refCmd.AddCommand(refShowCmd)
	refShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
