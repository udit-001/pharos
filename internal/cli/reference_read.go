package cli

import (
	"fmt"

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
		return runRead(cmd, readSpec[db.Reference]{
			fetch:    func(ws *db.WorkspaceStore) ([]db.Reference, error) { return ws.GetRefs() },
			errLabel: "failed to get references",
			findItem: func(items []db.Reference, key string) (*db.Reference, error) {
				for i := range items {
					if items[i].Slug == key {
						return &items[i], nil
					}
				}
				return nil, nil
			},
			keyName: "reference",
			jsonOut: func(item db.Reference, ws db.Workspace, wsStore *db.WorkspaceStore) map[string]any {
				return map[string]any{
					"slug":      item.Slug,
					"title":     item.Title,
					"filename":  item.Filename,
					"summary":   item.Summary,
					"createdAt": item.CreatedAt,
					"updatedAt": item.UpdatedAt,
				}
			},
			plainOut: func(item db.Reference, ws db.Workspace, wsStore *db.WorkspaceStore) {
				fmt.Printf("  Reference: %s\n", item.Slug)
				fmt.Printf("  Title: %s\n", item.Title)
				fmt.Printf("  File: %s\n", item.Filename)
				fmt.Printf("  Summary: %s\n", item.Summary)
				fmt.Printf("  Created: %s\n", item.CreatedAt)
				fmt.Printf("  Updated: %s\n", item.UpdatedAt)
			},
			bodyPath: func(wsStore *db.WorkspaceStore, item db.Reference) string {
				return wsStore.Layout().RefPath(item.Filename)
			},
		}, args[0])
	},
}

func init() {
	refCmd.AddCommand(refReadCmd)
	refReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	refReadCmd.Flags().Bool("meta-only", false, "Show metadata only, skip body content")
}
