package cli

import (
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var recordShowCmd = &cobra.Command{
	Use:   "show <seq>",
	Short: "Get a learning record's dashboard URL",
	Long: `Print the dashboard URL for viewing a learning record.

Examples:
  pharos record show 5
  pharos record show 5 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := parseSeq(args[0])
		if err != nil {
			return err
		}
		return runShow(cmd, showSpec{
			urlPath: func(wsName string) string {
				return urls.Record(wsName, n)
			},
			label: "record",
		})
	},
}

func init() {
	recordCmd.AddCommand(recordShowCmd)
	recordShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
