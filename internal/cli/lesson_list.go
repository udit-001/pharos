package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var lessonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List lessons in a workspace",
	Long: `List all lessons for a workspace, ordered by sequence number, with any
linked quiz shown per lesson.

Examples:
  pharos lesson list --workspace "sql-for-research"
  pharos lesson list
  pharos lesson list --search "join"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		search, _ := cmd.Flags().GetString("search")
		var lessons []db.Lesson
		if search != "" {
			lessons, err = wsStore.SearchLessons(search)
		} else {
			lessons, err = wsStore.GetLessons()
		}
		if err != nil {
			return formatError("failed to list lessons", err)
		}

		// Build the lesson -> quiz-slugs map once (reverse of Quiz.LessonSlug)
		// so each row can show whether the lesson's skill is practiced.
		quizByLesson := map[string][]string{}
		if quizzes, qerr := wsStore.GetQuizzes(); qerr == nil {
			for _, q := range quizzes {
				if q.LessonSlug != nil {
					quizByLesson[*q.LessonSlug] = append(quizByLesson[*q.LessonSlug], q.Slug)
				}
			}
		}

		if jsonEnabled(cmd) {
			out := make([]map[string]any, 0, len(lessons))
			for _, l := range lessons {
				entry := map[string]any{
					"slug":    l.Slug,
					"title":   l.Title,
					"summary": l.Summary,
				}
				if slugs, ok := quizByLesson[l.Slug]; ok {
					entry["quizzes"] = slugs
				}
				out = append(out, entry)
			}
			printJSON(out)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Workspace: %s\n\n", ws.DisplayName())

		if len(lessons) == 0 {
			fmt.Printf("  No lessons yet.\n")
			fmt.Printf("  Use 'pharos lesson create \"Title\" --workspace %q --body-file <path>' to add one.\n", ws.Name)
			fmt.Println()
			return nil
		}

		rows := make([][]string, 0, len(lessons))
		for _, l := range lessons {
			rows = append(rows, []string{
				truncate(l.Title, 40),
				l.Slug,
				quizCell(quizByLesson[l.Slug]),
			})
		}
		fmt.Println(formatTable([]string{"Title", "Slug", "Quiz"}, rows))
		fmt.Println()
		return nil
	},
}

// quizCell renders a lesson's linked quizzes for the list table: the first
// slug with " +N" when more follow, or "—" when none.
func quizCell(slugs []string) string {
	if len(slugs) == 0 {
		return "—"
	}
	if len(slugs) == 1 {
		return truncate(slugs[0], 18)
	}
	return fmt.Sprintf("%s +%d", truncate(slugs[0], 14), len(slugs)-1)
}

func init() {
	lessonCmd.AddCommand(lessonListCmd)
	lessonListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonListCmd.Flags().String("search", "", "Search lessons by title or summary")
}
