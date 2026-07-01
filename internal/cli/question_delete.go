package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a question from the workspace",
	Long: `Delete a question by slug. Blocks if any quiz references the
question — remove it from those quizzes first with
'pharos quiz revise <quiz-slug> --items ...'.

Attempt history for the question is removed with it (cascade).

Examples:
  pharos question delete bad-question --workspace "sql"
  pharos question delete duplicate-q -w "sql"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		if err := wsStore.DeleteQuestion(slug); err != nil {
			if errors.Is(err, db.ErrQuestionInUse) {
				quizzes, qerr := wsStore.GetQuizzes()
				if qerr != nil {
					return formatError("question in use (could not list quizzes for details)", err)
				}
				var refs []string
				for _, qz := range quizzes {
					if containsSlug(qz.Items, slug) {
						refs = append(refs, qz.Slug)
					}
				}
				if len(refs) == 0 {
					return fmt.Errorf("question %q is used by a quiz but the referencing quiz could not be found", slug)
				}
				return fmt.Errorf("question %q is used by quiz: %s\n  Remove it first:\n  pharos quiz revise %s --items <slugs-without-%s>",
					slug, strings.Join(refs, ", "), refs[0], slug)
			}
			return formatError("failed to delete question", err)
		}

		if jsonOut {
			printJSON(map[string]string{"status": "deleted", "slug": slug})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Question deleted: %s\n", slug)
		fmt.Println()
		return nil
	},
}

func init() {
	questionCmd.AddCommand(questionDeleteCmd)
	questionDeleteCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}

// containsSlug checks whether a JSON array of slugs contains the given slug.
func containsSlug(itemsJSON, slug string) bool {
	var items []string
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return false
	}
	for _, s := range items {
		if s == slug {
			return true
		}
	}
	return false
}
