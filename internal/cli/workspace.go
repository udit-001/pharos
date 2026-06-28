package cli

import "github.com/spf13/cobra"

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage learning workspaces",
	Long: `List and manage learning workspaces.

Examples:
  pharos workspace list
  pharos workspace stats`,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
