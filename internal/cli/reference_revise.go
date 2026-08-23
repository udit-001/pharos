package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var refReviseCmd = &cobra.Command{
	Use:   "revise <slug>",
	Short: "Revise an existing reference document",
	Long: `Overwrite a reference's content in place. The slug and filename are unchanged.

At least one of --body-file, --title, or --summary is required. A
metadata-only revise (--title/--summary without --body-file) leaves the
reference file and its search text untouched.

Examples:
  pharos reference revise sql-syntax --body-file /tmp/new-ref.html
  pharos reference revise sql-syntax --body-file /tmp/new-ref.html --title "Updated Title"
  pharos reference revise sql-syntax --title "Renamed" --summary "Metadata-only revise"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		in, err := parseReviseInputs(cmd, fmt.Sprintf("pharos reference revise %s --title \"New Title\" --summary \"Updated summary\"", slug))
		if err != nil {
			return err
		}

		if err := wsStore.ReviseRef(slug, in.body, in.title, in.summary); err != nil {
			return formatError("failed to revise reference", err)
		}

		ws := wsStore.Workspace()
		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyPageChanged(ws.Name, "ref", 0, slug)

		if jsonEnabled(cmd) {
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
	refReviseCmd.Flags().String("body-file", "", "Read reference HTML content from a file (replaces the existing body)")
	refReviseCmd.Flags().String("title", "", "Update the reference title")
	refReviseCmd.Flags().String("summary", "", "Update the reference summary")
}
