package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quizzes in a workspace",
	Long: `List all quizzes for a workspace, with the best score from completed
attempts.

Use --weak to sort by weakness: never-attempted quizzes first, then by
best-score ratio ascending (lowest accuracy first). This is the workspace's
skill-area weakness signal — use it to decide what to practice or teach next.

Examples:
  pharos quiz list --workspace "sql-for-research"
  pharos quiz list --workspace "sql-for-research" --json
  pharos quiz list --workspace "sql-for-research" --weak --limit 5`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		weak, _ := cmd.Flags().GetBool("weak")
		if weak {
			return runWeakQuizList(cmd)
		}
		return runList(cmd, listSpec[db.QuizScore]{
			fetch: func(ws *db.WorkspaceStore, search string) ([]db.QuizScore, error) {
				return ws.GetQuizScores()
			},
			errLabel:   "failed to list quizzes",
			emptyMsg:   "No quizzes yet.",
			createHint: `pharos quiz create "Title" --workspace %q --items "slug1,slug2"`,
			headers:    []string{"Slug", "Title", "Items", "Best", "Lesson"},
			buildRow:   quizScoreRow,
		})
	},
}

// runWeakQuizList sorts quizzes by weakness: never-attempted first (most
// urgent to assess), then by best-score ratio ascending. Mirrors
// runWeakList for questions.
func runWeakQuizList(cmd *cobra.Command) error {
	s := mustStore(cmd)
	wsName, _ := cmd.Flags().GetString("workspace")
	wsStore, err := resolveWorkspace(s, wsName)
	if err != nil {
		return err
	}
	ws := wsStore.Workspace()

	limit, _ := cmd.Flags().GetInt("limit")
	results, err := wsStore.GetQuizScores()
	if err != nil {
		return formatError("failed to query quiz scores", err)
	}

	sort.SliceStable(results, func(i, j int) bool {
		ai, aj := results[i], results[j]
		if ai.Attempted != aj.Attempted {
			return !ai.Attempted // never-attempted sorts first
		}
		if !ai.Attempted {
			return ai.Slug < aj.Slug // both unattempted: stable by slug
		}
		ri := ratio(ai.BestScore, ai.BestTotal)
		rj := ratio(aj.BestScore, aj.BestTotal)
		if ri != rj {
			return ri < rj // lowest accuracy first
		}
		return ai.Slug < aj.Slug
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	if jsonOut {
		if results == nil {
			results = []db.QuizScore{}
		}
		printJSON(results)
		return nil
	}

	fmt.Println()
	fmt.Printf("  Workspace: %s\n\n", ws.DisplayName())

	if len(results) == 0 {
		fmt.Printf("  No quizzes yet.\n")
		fmt.Printf("  Use 'pharos quiz create \"Title\" --workspace %q --items \"slug1,slug2\"' to add one.\n", ws.Name)
		fmt.Println()
		return nil
	}

	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, quizScoreRow(r))
	}
	fmt.Println(formatTable([]string{"Slug", "Title", "Items", "Best", "Lesson"}, rows))
	fmt.Println()
	return nil
}

// quizScoreRow renders one QuizScore as a table row. Shared by the plain list
// and the weak list so the row shape stays in one place.
func quizScoreRow(s db.QuizScore) []string {
	items, _ := s.ParseItems()
	best := "—"
	if s.Attempted {
		best = fmt.Sprintf("%d/%d", s.BestScore, s.BestTotal)
	}
	return []string{
		s.Slug,
		truncate(s.Title, 40),
		fmt.Sprintf("%d", len(items)),
		best,
		lessonRef(s.LessonSeq),
	}
}

// ratio returns correct/total, guarding against an empty quiz (total 0).
func ratio(correct, total int) float64 {
	if total < 1 {
		return 0
	}
	return float64(correct) / float64(total)
}

func init() {
	quizCmd.AddCommand(quizListCmd)
	quizListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	quizListCmd.Flags().Bool("weak", false, "Sort by weakness: never-attempted first, then by best-score ratio ascending")
	quizListCmd.Flags().Int("limit", 0, "Max results (0 = all)")
}
