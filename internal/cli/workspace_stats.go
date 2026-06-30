package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var workspaceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show learning statistics across all workspaces",
	Long: `Show summary of learning progress across all workspaces.

Examples:
  pharos workspace stats
  pharos workspace stats --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		workspaces, err := s.GetWorkspaces()
		if err != nil {
			return formatError("failed to get stats", err)
		}

		// Totals sums lesson/record/quiz/ref counts across workspaces in one
		// place — the command doesn't re-sum. GetWorkspaces already populates
		// the per-workspace QuizCount the helper reads.
		totals := db.Totals(workspaces)

		if jsonOut {
			printJSON(map[string]any{
				"totalWorkspaces": totals.Workspaces,
				"totalLessons":    totals.Lessons,
				"totalRecords":    totals.Records,
				"totalQuizzes":    totals.Quizzes,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Total workspaces:    %d\n", totals.Workspaces)
		fmt.Printf("  Total lessons:       %d\n", totals.Lessons)
		fmt.Printf("  Total learning recs: %d\n", totals.Records)
		fmt.Printf("  Total quizzes:       %d\n", totals.Quizzes)
		fmt.Println()

		if totals.Workspaces > 0 {
			fmt.Println("  By workspace:")
			for _, w := range workspaces {
				fmt.Printf("    %-25s L:%3d %s  R:%3d %s  Q:%3d %s\n",
					w.DisplayName(),
					w.LessonCount, countBar(w.LessonCount, totals.Lessons),
					w.RecordCount, countBar(w.RecordCount, totals.Records),
					w.QuizCount, countBar(w.QuizCount, totals.Quizzes),
				)
			}
			fmt.Println()
		}

		return nil
	},
}

// countBar returns a proportional block string for n out of total, up to 20
// blocks. Empty when total is zero. Shared across the lesson/record/quiz
// per-workspace lines so the bar shape lives in one place.
func countBar(n, total int) string {
	if total <= 0 {
		return ""
	}
	count := int(float64(n) / float64(total) * 100 / 5)
	b := ""
	for i := 0; i < count && i < 20; i++ {
		b += "█"
	}
	return b
}

func init() {
	workspaceCmd.AddCommand(workspaceStatsCmd)
}
