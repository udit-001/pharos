package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
				return fmt.Sprintf("/w/%s/records/%d", wsName, n)
			},
			label: "record",
		})
	},
}

func init() {
	recordCmd.AddCommand(recordShowCmd)
	recordShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
