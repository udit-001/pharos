package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var refReadCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Read a reference document's content and metadata",
	Long: `Print a reference's metadata and body content. Use --meta-only to skip the body.

Examples:
  pharos reference read sql-syntax
  pharos reference read sql-syntax --meta-only
  pharos reference read sql-syntax --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		refs, err := wsStore.GetRefs()
		if err != nil {
			return formatError("failed to get references", err)
		}

		var current *db.Reference
		for i := range refs {
			if refs[i].Slug == slug {
				current = &refs[i]
				break
			}
		}
		if current == nil {
			return fmt.Errorf("reference %q not found", slug)
		}

		metaOnly, _ := cmd.Flags().GetBool("meta-only")

		if jsonOut {
			result := map[string]any{
				"id":        current.ID,
				"slug":      current.Slug,
				"title":     current.Title,
				"filename":  current.Filename,
				"summary":   current.Summary,
				"createdAt": current.CreatedAt,
				"updatedAt": current.UpdatedAt,
				"workspace": ws.Name,
			}
			if !metaOnly {
				data, err := os.ReadFile(wsStore.Layout().RefPath(current.Filename))
				if err != nil {
					return fmt.Errorf("read reference file: %w", err)
				}
				result["body"] = string(data)
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Reference: %s\n", current.Slug)
		fmt.Printf("  Title: %s\n", current.Title)
		fmt.Printf("  File: %s\n", current.Filename)
		fmt.Printf("  Summary: %s\n", current.Summary)
		fmt.Printf("  Created: %s\n", current.CreatedAt)
		fmt.Printf("  Updated: %s\n", current.UpdatedAt)
		fmt.Println()

		if !metaOnly {
			data, err := os.ReadFile(wsStore.Layout().RefPath(current.Filename))
			if err != nil {
				return fmt.Errorf("read reference file: %w", err)
			}
			fmt.Println(string(data))
		}
		return nil
	},
}

func init() {
	refCmd.AddCommand(refReadCmd)
	refReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	refReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
