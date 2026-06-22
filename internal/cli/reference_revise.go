package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var refReviseCmd = &cobra.Command{
	Use:   "revise <slug>",
	Short: "Revise an existing reference document",
	Long: `Overwrite a reference's content in place. The slug and filename are unchanged.

Examples:
  pharos reference revise sql-syntax --body-file /tmp/new-ref.html
  pharos reference revise sql-syntax --body-file /tmp/new-ref.html --title "Updated Title"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
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

		if err := wsStore.ReviseRef(slug, string(data), titlePtr, summaryPtr); err != nil {
			return formatError("failed to revise reference", err)
		}

		if jsonOut {
			printJSON(map[string]string{"status": "revised", "slug": slug})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Reference revised: %s\n", slug)
		fmt.Println()
		return nil
	},
}

func init() {
	refCmd.AddCommand(refReviseCmd)
	refReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	refReviseCmd.Flags().String("body-file", "", "Read reference HTML content from a file (required)")
	refReviseCmd.Flags().String("title", "", "Update the reference title")
	refReviseCmd.Flags().String("summary", "", "Update the reference summary")
}
