package cli

import (
	"errors"
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

		// CreateRef owns the invariant: slugify, duplicate-slug check, file
		// write, body_text extraction, and the DB row. The CLI shrinks to
		// parse-and-call.
		created, err := wsStore.CreateRef(title, html)
		if err != nil {
			if errors.Is(err, db.ErrRefSlugExists) {
				slug := db.Slugify(title)
				return fmt.Errorf("reference with slug %q already exists\n  Use 'pharos reference revise %s' to update it", slug, slug)
			}
			return formatError("failed to save reference", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Reference created: %s\n", title)
		fmt.Printf("    File: %s\n", filepath.Join(ws.Path, created.Path))
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
