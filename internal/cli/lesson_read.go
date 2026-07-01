package cli

import (
	"fmt"
	"strings"

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
		return runRead(cmd, readSpec[db.Lesson]{
			fetch:    func(ws *db.WorkspaceStore) ([]db.Lesson, error) { return ws.GetLessons() },
			errLabel: "failed to get lessons",
			findItem: func(items []db.Lesson, key string) (*db.Lesson, error) {
				n, err := parseSeq(key)
				if err != nil {
					return nil, err
				}
				for i := range items {
					if items[i].SequenceNumber == n {
						return &items[i], nil
					}
				}
				return nil, nil
			},
			keyName: "lesson",
			jsonOut: func(item db.Lesson, ws db.Workspace, wsStore *db.WorkspaceStore) map[string]any {
				m := map[string]any{
					"sequenceNumber": item.SequenceNumber,
					"title":          item.Title,
					"filename":       item.Filename,
					"summary":        item.Summary,
					"createdAt":      item.CreatedAt,
					"updatedAt":      item.UpdatedAt,
				}
				linkedQuizzes, _ := wsStore.LessonContent().QuizzesForLesson(item.SequenceNumber)
				if len(linkedQuizzes) > 0 {
					slugs := make([]string, len(linkedQuizzes))
					for i, q := range linkedQuizzes {
						slugs[i] = q.Slug
					}
					m["quizzes"] = slugs
				}
				return m
			},
			plainOut: func(item db.Lesson, ws db.Workspace, wsStore *db.WorkspaceStore) {
				fmt.Printf("  Lesson #%d: %s\n", item.SequenceNumber, item.Title)
				fmt.Printf("  File: %s\n", item.Filename)
				fmt.Printf("  Summary: %s\n", item.Summary)
				fmt.Printf("  Created: %s\n", item.CreatedAt)
				fmt.Printf("  Updated: %s\n", item.UpdatedAt)
				linkedQuizzes, _ := wsStore.LessonContent().QuizzesForLesson(item.SequenceNumber)
				if len(linkedQuizzes) > 0 {
					slugs := make([]string, len(linkedQuizzes))
					for i, q := range linkedQuizzes {
						slugs[i] = q.Slug
					}
					fmt.Printf("  Quizzes: %s\n", strings.Join(slugs, ", "))
				}
			},
			bodyPath: func(wsStore *db.WorkspaceStore, item db.Lesson) string {
				return wsStore.Layout().LessonPath(item.Filename)
			},
		}, args[0])
	},
}

func init() {
	lessonCmd.AddCommand(lessonReadCmd)
	lessonReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
