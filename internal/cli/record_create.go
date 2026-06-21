package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/udit-001/pharos/internal/db"
	"github.com/spf13/cobra"
)

var recordCreateCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new learning record",
	Long: `Create a new learning record file in the workspace.

Learning records capture what the user has learned — non-obvious
lessons, key insights, and stated prior knowledge. They are
titled 0001-<dash-case-name>.md with sequential numbering.

Examples:
  learn record add "The connection sequence matters" --workspace "jump-start-a-car"
  learn record add "No prior CAM knowledge" --workspace "cell-adhesion"
  learn record add "Understood SELECT, WHERE, JOIN" --workspace "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		title := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		// Get next sequence number
		records, err := wsStore.GetRecords()
		if err != nil {
			return formatError("failed to get records", err)
		}
		seqNum := len(records) + 1

		// Create filename
		slug := slugify(title)
		filename := fmt.Sprintf("%04d-%s.md", seqNum, slug)
		recordPath := filepath.Join(ws.Path, "learning-records", filename)

		// Record body comes from --body-file (required) — no stub template.
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  Write the record markdown to a file, then: learn record add %q --workspace %q --body-file <path>", title, ws.Name)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}
		content := string(data)

		if err := os.WriteFile(recordPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write learning record: %w", err)
		}

		// Get summary from optional flag
		summary, _ := cmd.Flags().GetString("summary")

		// Save to database (WorkspaceID auto-set by the scoped store)
		created, err := wsStore.AddRecord(db.LearningRecord{
			Title:    title,
			Filename: filename,
			Path:    filepath.Join("learning-records", filename),
			Summary: summary,
		})
		if err != nil {
			return formatError("failed to save learning record", err)
		}

		_ = wsStore.Touch()

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Learning record created: %s\n", title)
		fmt.Printf("    File: %s\n", recordPath)
		fmt.Printf("    Workspace: %s\n", ws.Name)
		fmt.Println()

		return nil
	},
}

func init() {
	recordCmd.AddCommand(recordCreateCmd)
	recordCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	recordCreateCmd.Flags().String("summary", "", "Brief summary of what was learned")
	recordCreateCmd.Flags().String("body-file", "", "Read record markdown content from a file (required)")
}
