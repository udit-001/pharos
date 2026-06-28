package cli

import "github.com/spf13/cobra"

var glossaryCmd = &cobra.Command{
	Use:   "glossary",
	Short: "Manage workspace glossary terms",
	Long: `Display, add, and delete glossary terms for a workspace.

The glossary records canonical terminology for this workspace.

Examples:
  pharos glossary list
  pharos glossary create "<term>" "<definition>"
  pharos glossary delete "<term>"`,
}

func init() {
	rootCmd.AddCommand(glossaryCmd)
}
