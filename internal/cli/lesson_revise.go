package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var lessonReviseCmd = &cobra.Command{
	Use:   "revise <seq>",
	Short: "Revise an existing lesson",
	Long: `Overwrite a lesson's content in place. The sequence number and filename are unchanged.

Examples:
  pharos lesson revise 3 --body-file /tmp/new-lesson.html
  pharos lesson revise 3 --body-file /tmp/new-lesson.html --title "Updated Title"
  pharos lesson revise 3 --body-file /tmp/new-lesson.html --summary "Updated summary"`,
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

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required")
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		var titlePtr, summaryPtr *string
		if v, _ := cmd.Flags().GetString("title"); v != "" {
			titlePtr = &v
		}
		if v, _ := cmd.Flags().GetString("summary"); v != "" {
			summaryPtr = &v
		}

		if err := wsStore.ReviseLesson(seq, string(data), titlePtr, summaryPtr); err != nil {
			return formatError("failed to revise lesson", err)
		}

		if jsonOut {
			printJSON(map[string]string{"status": "revised", "sequence": fmt.Sprintf("%d", seq)})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Lesson #%d revised\n", seq)
		fmt.Println()
		return nil
	},
}

func init() {
	lessonCmd.AddCommand(lessonReviseCmd)
	lessonReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonReviseCmd.Flags().String("body-file", "", "Read lesson HTML content from a file (required)")
	lessonReviseCmd.Flags().String("title", "", "Update the lesson title")
	lessonReviseCmd.Flags().String("summary", "", "Update the lesson summary")
}
