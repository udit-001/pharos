package cli

import "github.com/spf13/cobra"

var quizCmd = &cobra.Command{
	Use:   "quiz",
	Short: "Manage quizzes in a workspace",
	Long: `Create and list quizzes.

A quiz is an ordered list of question slugs grouped under a title.
Quizzes are DB-only (no file on disk).

Examples:
  pharos quiz create "SQL Basics" --items "what-is-a-join,explain-mvcc" --workspace "sql-for-research"
  pharos quiz list --workspace "sql-for-research"`,
}

func init() {
	rootCmd.AddCommand(quizCmd)
}
