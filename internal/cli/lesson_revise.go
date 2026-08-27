package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var lessonReviseCmd = &cobra.Command{
	Use:   "revise <slug>",
	Short: "Revise an existing lesson",
	Long: `Overwrite a lesson's content in place. The slug is unchanged.

At least one of --body-file, --title, or --summary is required. A
metadata-only revise (--title/--summary without --body-file) leaves the
lesson file and its search text untouched.

Examples:
  pharos lesson revise sql-joins --body-file /tmp/new-lesson.html
  pharos lesson revise sql-joins --body-file /tmp/new-lesson.html --title "Updated Title"
  pharos lesson revise sql-joins --title "Renamed" --summary "Metadata-only revise"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		lesson, err := wsStore.GetLessonBySlug(slug)
		if err != nil {
			return formatError("lesson not found", err)
		}

		in, err := parseReviseInputs(cmd, fmt.Sprintf("pharos lesson revise %s --title \"New Title\" --summary \"Updated summary\"", slug))
		if err != nil {
			return err
		}

		if err := wsStore.ReviseLesson(lesson.SequenceNumber, in.body, in.title, in.summary); err != nil {
			return formatError("failed to revise lesson", err)
		}

		// Live-sync: refresh the sidebar (title may have changed) and
		// reload the lesson iframe if it's the one currently open.
		ws := wsStore.Workspace()
		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyPageChanged(ws.Name, "lesson", lesson.SequenceNumber, "")

		lessonURL := urls.Lesson(ws.Name, slug)

		if jsonEnabled(cmd) {
			printJSON(map[string]string{"status": "revised", "slug": slug, "url": lessonURL})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Lesson %q revised\n", slug)
		fmt.Printf("    URL: %s\n", lessonURL)
		fmt.Println()
		return nil
	},
}

func init() {
	lessonCmd.AddCommand(lessonReviseCmd)
	lessonReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonReviseCmd.Flags().String("body-file", "", "Read lesson HTML content from a file (replaces the existing body)")
	lessonReviseCmd.Flags().String("title", "", "Update the lesson title")
	lessonReviseCmd.Flags().String("summary", "", "Update the lesson summary")
}
