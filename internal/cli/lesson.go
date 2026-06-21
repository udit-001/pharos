package cli

import "github.com/spf13/cobra"

var lessonCmd = &cobra.Command{
	Use:   "lesson",
	Short: "Manage lessons in a workspace",
	Long: `Create, list, and manage lesson files.

Lessons are self-contained HTML files in the workspace's lessons/
directory. Each lesson teaches one tightly-scoped thing.

Examples:
  pharos lesson create "SQL Joins" --workspace "sql-for-research"
  pharos lesson list --workspace "sql-for-research"`,
}

func init() {
	rootCmd.AddCommand(lessonCmd)
}
