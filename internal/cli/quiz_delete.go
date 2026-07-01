package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a quiz from the workspace",
	Long: `Delete a quiz by slug. Blocks if the quiz has in-progress
attempts — wait for them to complete or be abandoned first. All
completed attempt history is removed with the quiz (cascade).

Examples:
  pharos quiz delete wrong-quiz --workspace "sql"
  pharos quiz delete duplicate-q -w "sql"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		if err := wsStore.DeleteQuiz(slug); err != nil {
			if errors.Is(err, db.ErrQuizHasInProgress) {
				return fmt.Errorf("quiz %q has in-progress attempts\n  Wait for them to complete, or abandon them first", slug)
			}
			return formatError("failed to delete quiz", err)
		}

		if jsonOut {
			printJSON(map[string]string{"status": "deleted", "slug": slug})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Quiz deleted: %s\n", slug)
		fmt.Println()
		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizDeleteCmd)
	quizDeleteCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
