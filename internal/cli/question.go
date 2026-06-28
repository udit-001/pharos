package cli

import "github.com/spf13/cobra"

var questionCmd = &cobra.Command{
	Use:   "question",
	Short: "Manage questions in a workspace",
	Long: `Create and list questions.

Questions are DB-only entities (no file on disk) used to build quizzes.
A question's mode determines its config shape: "choice" (multiple-choice
with a correct answer) or "recall" (self-graded reveal text).

Examples:
  pharos question create "What is a JOIN?" --workspace "sql-for-research" --mode choice --body-file /tmp/q.json
  pharos question list --workspace "sql-for-research"`,
}

func init() {
	rootCmd.AddCommand(questionCmd)
}
