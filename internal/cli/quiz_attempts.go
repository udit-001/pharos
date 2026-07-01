package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizAttemptsCmd = &cobra.Command{
	Use:   "attempts <slug>",
	Short: "Show a quiz's attempt history and trend",
	Long: `Print the completed attempts for a quiz in chronological order, each
scored against the quiz's current questions, with a trend summary (is
accuracy improving across retakes?). In-progress and abandoned attempts are
excluded — they have no score.

The per-attempt scores use the same computation as ` + "`quiz list`" + `'s
best-score column, so the trend reconciles with it: the best score is the
max of this series.

Examples:
  pharos quiz attempts sql-basics
  pharos quiz attempts sql-basics --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		quiz, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			return formatError("failed to find quiz", err)
		}
		history, err := wsStore.GetQuizAttemptHistory(quiz.ID)
		if err != nil {
			return formatError("failed to get attempt history", err)
		}

		if jsonOut {
			if history == nil {
				history = []db.QuizAttemptScore{}
			}
			printJSON(history)
			return nil
		}

		fmt.Println()
		fmt.Printf("  %s\n", quiz.Title)
		fmt.Printf("  Slug: %s\n", quiz.Slug)
		fmt.Printf("  Lesson: %s\n", lessonRef(quiz.LessonSeq))
		fmt.Println()

		if len(history) == 0 {
			fmt.Printf("  No completed attempts yet.\n")
			fmt.Printf("  Use 'pharos quiz show %s' to take the quiz in the dashboard.\n", slug)
			fmt.Println()
			return nil
		}

		rows := make([][]string, 0, len(history))
		for _, h := range history {
			rows = append(rows, []string{
				formatDateShort(h.CompletedAt),
				fmt.Sprintf("%d/%d", h.Correct, h.Total),
				fmt.Sprintf("%.0f%%", ratio(h.Correct, h.Total)*100),
			})
		}
		fmt.Println(formatTable([]string{"Completed", "Score", "Accuracy"}, rows))
		fmt.Println()
		fmt.Printf("  Trend: %s\n", trendLabel(history))
		fmt.Println()
		return nil
	},
}

// trendLabel summarises the accuracy trajectory by comparing the average of
// the earlier half of completed attempts against the recent half. Smoother
// than first-vs-last, which misses mid-series volatility. The table is the
// real data; this is a convenience hint.
func trendLabel(history []db.QuizAttemptScore) string {
	if len(history) < 2 {
		return "single attempt — no trend yet"
	}
	mid := len(history) / 2
	earlier := avgAccuracy(history[:mid])
	recent := avgAccuracy(history[mid:])
	switch {
	case recent > earlier:
		return fmt.Sprintf("improving (%.0f%% → %.0f%%)", earlier*100, recent*100)
	case recent < earlier:
		return fmt.Sprintf("declining (%.0f%% → %.0f%%)", earlier*100, recent*100)
	default:
		return fmt.Sprintf("flat (%.0f%%)", recent*100)
	}
}

// avgAccuracy is the mean per-attempt accuracy ratio over a slice.
func avgAccuracy(history []db.QuizAttemptScore) float64 {
	if len(history) == 0 {
		return 0
	}
	sum := 0.0
	for _, h := range history {
		sum += ratio(h.Correct, h.Total)
	}
	return sum / float64(len(history))
}

func init() {
	quizCmd.AddCommand(quizAttemptsCmd)
	quizAttemptsCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
