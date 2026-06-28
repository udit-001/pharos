package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List questions in a workspace",
	Long: `List all questions for a workspace.

Use --weak to sort by accuracy ascending (questions the learner struggles
with most), based on completed quiz attempts only. Questions with no attempts
sort first.

Examples:
  pharos question list --workspace "sql-for-research"
  pharos question list --workspace "sql-for-research" --json
  pharos question list --workspace "sql-for-research" --weak --json --limit 5`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		weak, _ := cmd.Flags().GetBool("weak")
		if weak {
			return runWeakList(cmd)
		}
		return runList(cmd, listSpec[db.Question]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.Question, error) {
				return ws.GetQuestions()
			},
			errLabel:   "failed to list questions",
			emptyMsg:   "No questions yet.",
			createHint: `pharos question create "Title" --workspace %q --mode choice --body-file <path>`,
			headers:    []string{"Slug", "Title", "Mode"},
			buildRow: func(q db.Question) []string {
				return []string{
					q.Slug,
					truncate(q.Title, 40),
					q.Mode,
				}
			},
		})
	},
}

func runWeakList(cmd *cobra.Command) error {
	s := mustStore(cmd)
	wsName, _ := cmd.Flags().GetString("workspace")
	wsStore, err := resolveWorkspace(s, wsName)
	if err != nil {
		return err
	}
	ws := wsStore.Workspace()

	limit, _ := cmd.Flags().GetInt("limit")
	results, err := wsStore.GetWeakQuestions(limit)
	if err != nil {
		return formatError("failed to query weak questions", err)
	}

	if jsonOut {
		if results == nil {
			results = []db.WeakQuestionResult{}
		}
		printJSON(results)
		return nil
	}

	fmt.Println()
	fmt.Printf("  Workspace: %s\n\n", ws.DisplayName())

	if len(results) == 0 {
		fmt.Printf("  No questions yet.\n")
		fmt.Printf("  Use 'pharos question create \"Title\" --workspace %q --mode choice --body-file <path>' to add one.\n", ws.Name)
		fmt.Println()
		return nil
	}

	rows := make([][]string, 0, len(results))
	for _, r := range results {
		acc := "—"
		if r.HasData {
			acc = fmt.Sprintf("%d/%d", r.Correct, r.Total)
		}
		rows = append(rows, []string{
			r.Slug,
			truncate(r.Title, 40),
			r.Mode,
			acc,
		})
	}
	fmt.Println(formatTable([]string{"Slug", "Title", "Mode", "Accuracy"}, rows))
	fmt.Println()
	return nil
}

func init() {
	questionCmd.AddCommand(questionListCmd)
	questionListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	questionListCmd.Flags().Bool("weak", false, "Sort by accuracy ascending (completed attempts only)")
	questionListCmd.Flags().Int("limit", 0, "Max results (0 = all)")
}
