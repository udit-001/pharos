package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var recordSupersedeCmd = &cobra.Command{
	Use:   "supersede <seq>",
	Short: "Supersede a learning record with new understanding",
	Long: `Atomically create a new learning record and mark the old one as superseded.

This is the ADR-style way to revise a learning record: you don't edit
the old one, you supersede it with updated thinking.

Examples:
  pharos record supersede 3 --title "Revised understanding" --body-file /tmp/new.md
  pharos record supersede 3 --title "Updated insight" --body-file /tmp/new.md --summary "Summary"`,
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

		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			return fmt.Errorf("--title is required")
		}

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required")
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		summary, _ := cmd.Flags().GetString("summary")

		created, old, err := wsStore.SupersedeRecord(seq, title, string(data), summary)
		if err != nil {
			return formatError("failed to supersede record", err)
		}

		if jsonOut {
			printJSON(map[string]any{
				"created":    created,
				"superseded": old,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Record created: %s (#%d)\n", created.Title, created.SequenceNumber)
		fmt.Printf("  ✓ Record #%d superseded by #%d\n", seq, created.SequenceNumber)
		fmt.Println()
		return nil
	},
}

func init() {
	recordCmd.AddCommand(recordSupersedeCmd)
	recordSupersedeCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	recordSupersedeCmd.Flags().String("title", "", "Title for the new record (required)")
	recordSupersedeCmd.Flags().String("body-file", "", "Read record markdown content from a file (required)")
	recordSupersedeCmd.Flags().String("summary", "", "Brief summary of the new record")
}
