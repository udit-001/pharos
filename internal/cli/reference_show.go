package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var refShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show a reference in the dashboard",
	Long: `Print the dashboard URL for a reference document. The dashboard must be running (use 'pharos start').

Examples:
  pharos reference show sql-syntax
  pharos reference show sql-syntax --workspace "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		// TODO: start dashboard if needed (PID file logic)
		url := "http://127.0.0.1:9090" + urls.Ref(ws.Name, slug)

		if jsonOut {
			printJSON(map[string]string{"url": url})
			return nil
		}

		fmt.Println()
		fmt.Printf("  View reference %s at: %s\n", slug, url)
		fmt.Printf("  Dashboard must be running (use 'pharos start').\n")
		fmt.Println()
		return nil
	},
}

func init() {
	refCmd.AddCommand(refShowCmd)
	refShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
