package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var lessonReviseCmd = &cobra.Command{
	Use:   "revise <seq>",
	Short: "Revise an existing lesson",
	Long: `Overwrite a lesson's content in place. The sequence number and filename are unchanged.

At least one of --body-file, --title, or --summary is required. A
metadata-only revise (--title/--summary without --body-file) leaves the
lesson file and its search text untouched.

Examples:
  pharos lesson revise 3 --body-file /tmp/new-lesson.html
  pharos lesson revise 3 --body-file /tmp/new-lesson.html --title "Updated Title"
  pharos lesson revise 3 --title "Renamed" --summary "Metadata-only revise"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		seq, err := parseSeq(args[0])
		if err != nil {
			return err
		}
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		in, err := parseReviseInputs(cmd, fmt.Sprintf("pharos lesson revise %d --title \"New Title\" --summary \"Updated summary\"", seq))
		if err != nil {
			return err
		}

		if err := wsStore.ReviseLesson(seq, in.body, in.title, in.summary); err != nil {
			return formatError("failed to revise lesson", err)
		}

		// Live-sync: refresh the sidebar (title may have changed) and
		// reload the lesson iframe if it's the one currently open.
		ws := wsStore.Workspace()
		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyPageChanged(ws.Name, "lesson", seq, "")

		lessonURL := urls.Lesson(ws.Name, seq)

		if jsonEnabled(cmd) {
			printJSON(map[string]string{"status": "revised", "sequence": fmt.Sprintf("%d", seq), "url": lessonURL})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Lesson #%d revised\n", seq)
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
