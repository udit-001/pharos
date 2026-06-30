package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var recordReadCmd = &cobra.Command{
	Use:   "read <seq>",
	Short: "Read a learning record's content and metadata",
	Long: `Print a learning record's metadata and body content. Use --meta-only to skip the body.

Examples:
  pharos record read 5
  pharos record read 5 --meta-only
  pharos record read 5 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRead(cmd, readSpec[db.LearningRecord]{
			fetch:    func(ws *db.WorkspaceStore) ([]db.LearningRecord, error) { return ws.GetRecords() },
			errLabel: "failed to get records",
			findItem: func(items []db.LearningRecord, key string) (*db.LearningRecord, error) {
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
			keyName: "record",
			jsonOut: func(item db.LearningRecord, ws db.Workspace, wsStore *db.WorkspaceStore) map[string]any {
				return map[string]any{
					"sequenceNumber": item.SequenceNumber,
					"title":          item.Title,
					"filename":       item.Filename,
					"status":         item.Status,
					"summary":        item.Summary,
					"createdAt":      item.CreatedAt,
					"updatedAt":      item.UpdatedAt,
				}
			},
			plainOut: func(item db.LearningRecord, ws db.Workspace, wsStore *db.WorkspaceStore) {
				fmt.Printf("  Record #%d: %s\n", item.SequenceNumber, item.Title)
				fmt.Printf("  Status: %s\n", item.Status)
				fmt.Printf("  File: %s\n", item.Filename)
				fmt.Printf("  Summary: %s\n", item.Summary)
				fmt.Printf("  Created: %s\n", item.CreatedAt)
				fmt.Printf("  Updated: %s\n", item.UpdatedAt)
			},
			bodyPath: func(wsStore *db.WorkspaceStore, item db.LearningRecord) string {
				return wsStore.Layout().RecordPath(item.Filename)
			},
		}, args[0])
	},
}

func init() {
	recordCmd.AddCommand(recordReadCmd)
	recordReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	recordReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
