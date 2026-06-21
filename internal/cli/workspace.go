package cli

import "github.com/spf13/cobra"

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage learning workspaces",
	Long: `List, open, and view stats for learning workspaces.

Examples:
  learn workspace list
  learn workspace open "sql-for-research"
  learn workspace stats`,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
