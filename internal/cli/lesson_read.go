package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var lessonReadCmd = &cobra.Command{
	Use:   "read <seq>",
	Short: "Read a lesson's content and metadata",
	Long: `Print a lesson's metadata and body content. Use --meta-only to skip the body.

Examples:
  pharos lesson read 3
  pharos lesson read 3 --meta-only
  pharos lesson read 3 --json`,
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
		ws := wsStore.Workspace()

		lessons, err := wsStore.GetLessons()
		if err != nil {
			return formatError("failed to get lessons", err)
		}

		var current *db.Lesson
		for i := range lessons {
			if lessons[i].SequenceNumber == seq {
				current = &lessons[i]
				break
			}
		}
		if current == nil {
			return fmt.Errorf("lesson #%d not found", seq)
		}

		metaOnly, _ := cmd.Flags().GetBool("meta-only")

		if jsonOut {
			result := map[string]any{
				"id":              current.ID,
				"sequenceNumber":  current.SequenceNumber,
				"title":           current.Title,
				"filename":        current.Filename,
				"summary":         current.Summary,
				"createdAt":       current.CreatedAt,
				"updatedAt":       current.UpdatedAt,
				"workspace":       ws.Name,
			}
			if !metaOnly {
				data, err := os.ReadFile(wsStore.Layout().LessonPath(current.Filename))
				if err != nil {
					return fmt.Errorf("read lesson file: %w", err)
				}
				result["body"] = string(data)
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Lesson #%d: %s\n", current.SequenceNumber, current.Title)
		fmt.Printf("  File: %s\n", current.Filename)
		fmt.Printf("  Summary: %s\n", current.Summary)
		fmt.Printf("  Created: %s\n", current.CreatedAt)
		fmt.Printf("  Updated: %s\n", current.UpdatedAt)
		fmt.Println()

		if !metaOnly {
			data, err := os.ReadFile(wsStore.Layout().LessonPath(current.Filename))
			if err != nil {
				return fmt.Errorf("read lesson file: %w", err)
			}
			fmt.Println(string(data))
		}
		return nil
	},
}

func init() {
	lessonCmd.AddCommand(lessonReadCmd)
	lessonReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
