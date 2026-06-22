package cli

import (
	"fmt"
	"os"

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

		records, err := wsStore.GetRecords()
		if err != nil {
			return formatError("failed to get records", err)
		}

		var current *db.LearningRecord
		for i := range records {
			if records[i].SequenceNumber == seq {
				current = &records[i]
				break
			}
		}
		if current == nil {
			return fmt.Errorf("record #%d not found", seq)
		}

		metaOnly, _ := cmd.Flags().GetBool("meta-only")

		if jsonOut {
			result := map[string]any{
				"id":              current.ID,
				"sequenceNumber":  current.SequenceNumber,
				"title":           current.Title,
				"filename":        current.Filename,
				"status":          current.Status,
				"summary":         current.Summary,
				"createdAt":       current.CreatedAt,
				"updatedAt":       current.UpdatedAt,
				"workspace":       ws.Name,
			}
			if !metaOnly {
				data, err := os.ReadFile(wsStore.Layout().RecordPath(current.Filename))
				if err != nil {
					return fmt.Errorf("read record file: %w", err)
				}
				result["body"] = string(data)
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Record #%d: %s\n", current.SequenceNumber, current.Title)
		fmt.Printf("  Status: %s\n", current.Status)
		fmt.Printf("  File: %s\n", current.Filename)
		fmt.Printf("  Summary: %s\n", current.Summary)
		fmt.Printf("  Created: %s\n", current.CreatedAt)
		fmt.Printf("  Updated: %s\n", current.UpdatedAt)
		fmt.Println()

		if !metaOnly {
			data, err := os.ReadFile(wsStore.Layout().RecordPath(current.Filename))
			if err != nil {
				return fmt.Errorf("read record file: %w", err)
			}
			fmt.Println(string(data))
		}
		return nil
	},
}

func init() {
	recordCmd.AddCommand(recordReadCmd)
	recordReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	recordReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
