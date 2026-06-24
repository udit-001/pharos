package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var refCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new reference document",
	Long: `Create a new reference HTML file in the workspace's reference/ directory.

Examples:
  pharos reference create "SQL Join Cheat Sheet" --workspace "sql-for-research"
  pharos reference create "Jumper Cable Steps" --workspace "jump-start-a-car"`,
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

		// Get existing refs to check for duplicate slug
		refs, err := wsStore.GetRefs()
		if err != nil {
			return formatError("failed to get references", err)
		}

		slug := db.Slugify(title)
		for _, r := range refs {
			if r.Slug == slug {
				return fmt.Errorf("reference with slug %q already exists\n  Use 'pharos reference revise %s' to update it", slug, slug)
			}
		}

		filename := slug + ".html"
		refPath := filepath.Join(ws.Path, "reference", filename)

		// Reference content comes from --body-file (required) — no stub template.
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  Write the reference HTML to a file, then: pharos reference create %q --workspace %q --body-file <path>", title, ws.Name)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}
		html := string(data)

		if err := os.WriteFile(refPath, []byte(html), 0644); err != nil {
			return fmt.Errorf("write reference file: %w", err)
		}

		// Save to database (WorkspaceID auto-set by the scoped store)
		created, err := wsStore.AddRef(db.Reference{
			Title:    title,
			Slug:     slug,
			Filename: filename,
			Path:     filepath.Join("reference", filename),
		})
		if err != nil {
			return formatError("failed to save reference", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Reference created: %s\n", title)
		fmt.Printf("    File: %s\n", refPath)
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

func init() {
	refCmd.AddCommand(refCreateCmd)
	refCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	refCreateCmd.Flags().String("body-file", "", "Read reference HTML content from a file (required)")
}
