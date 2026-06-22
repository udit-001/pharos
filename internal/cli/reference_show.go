package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var refShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show a reference in the dashboard",
	Long: `Open a reference document in the web dashboard. Starts the dashboard if not running.

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
		url := fmt.Sprintf("http://127.0.0.1:9090/workspace/%s/ref/%s", urlPathEscapeCLI(ws.Name), urlPathEscapeCLI(slug))

		if jsonOut {
			printJSON(map[string]string{"url": url})
			return nil
		}

		fmt.Println()
		fmt.Printf("  View reference %s at: %s\n", slug, url)
		fmt.Println()
		return nil
	},
}

func init() {
	refCmd.AddCommand(refShowCmd)
	refShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
