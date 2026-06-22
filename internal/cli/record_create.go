package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var recordCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new learning record",
	Long: `Create a new learning record file in the workspace.

Learning records capture what the user has learned — non-obvious
lessons, key insights, and stated prior knowledge.

Examples:
  pharos record create "The connection sequence matters" --workspace "jump-start-a-car"
  pharos record create "No prior CAM knowledge" --workspace "cell-adhesion"
  pharos record create "Understood SELECT, WHERE, JOIN" --workspace "sql-for-research"`,
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

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  Write the record markdown to a file, then: pharos record create %q --workspace %q --body-file <path>", title, ws.Name)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		summary, _ := cmd.Flags().GetString("summary")

		created, err := wsStore.CreateRecord(title, string(data), summary)
		if err != nil {
			return formatError("failed to create learning record", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Learning record created: %s (#%d)\n", title, created.SequenceNumber)
		fmt.Printf("    File: %s\n", wsStore.Layout().RecordPath(created.Filename))
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
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
