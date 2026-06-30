package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var recordShowCmd = &cobra.Command{
	Use:   "show <seq>",
	Short: "Show a learning record in the dashboard",
	Long: `Print the dashboard URL for a learning record. The dashboard must be running (use 'pharos start').

Examples:
  pharos record show 5
  pharos record show 5 --workspace "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		seq, err := parseSeq(args[0])
		if err != nil {
			return err
		}
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		// TODO: start dashboard if needed (PID file logic)
		url := "http://127.0.0.1:9090" + urls.Record(ws.Name, seq)

		if jsonOut {
			printJSON(map[string]string{"url": url})
			return nil
		}

		fmt.Println()
		fmt.Printf("  View record #%d at: %s\n", seq, url)
		fmt.Printf("  Dashboard must be running (use 'pharos start').\n")
		fmt.Println()
		return nil
	},
}

func init() {
	recordCmd.AddCommand(recordShowCmd)
	recordShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
