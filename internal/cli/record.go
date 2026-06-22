package cli

import "github.com/spf13/cobra"

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Manage learning records in a workspace",
	Long: `Create, list, and manage learning records.

Learning records are ADR-style documents in the workspace's
learning-records/ directory. They capture non-obvious lessons,
key insights, and stated prior knowledge that will steer
future sessions.

Examples:
  pharos record create "The connection sequence matters" --workspace "jump-start-a-car"
  pharos record list --workspace "sql-for-research"`,
}

func init() {
	rootCmd.AddCommand(recordCmd)
}
