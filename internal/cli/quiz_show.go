package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var quizShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show a quiz in the dashboard",
	Long: `Print the dashboard URL for a quiz. The dashboard must be running (use 'pharos start').

Examples:
  pharos quiz show sql-basics
  pharos quiz show sql-basics --workspace "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		// TODO: start dashboard if needed (PID file logic)
		url := "http://127.0.0.1:9090" + urls.Quiz(ws.Name, slug)

		if jsonOut {
			printJSON(map[string]string{"url": url})
			return nil
		}

		fmt.Println()
		fmt.Printf("  View quiz %s at: %s\n", slug, url)
		// Surface the lesson link even though the dashboard doesn't render it yet.
		if quiz, err := wsStore.GetQuizBySlug(slug); err == nil {
			fmt.Printf("  Lesson: %s\n", lessonRef(quiz.LessonSeq))
		}
		fmt.Printf("  Dashboard must be running (use 'pharos start').\n")
		fmt.Println()
		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizShowCmd)
	quizShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
