package cli

import "github.com/spf13/cobra"

var refCmd = &cobra.Command{
	Use:   "reference",
	Short: "Manage reference documents in a workspace",
	Long: `Create and list reference documents (cheat sheets).

References are the compressed essence of lessons — syntax
references, algorithms, glossaries. They live in the reference/
directory and are designed for quick lookup.

Examples:
  pharos reference create "SQL Cheat Sheet" --workspace "sql-for-research"
  pharos reference list --workspace "sql-for-research"`,
}

func init() {
	rootCmd.AddCommand(refCmd)
}
