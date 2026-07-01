package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new learning workspace",
	Long: `Create a new learning workspace.

The workspace is a directory under your data directory's workspaces/ containing:
  MISSION.md          — Why you're learning this topic
  RESOURCES.md        — Curated sources and communities
  NOTES.md            — Preferences and working notes
  lessons/            — Self-contained lesson HTML files
  learning-records/   — ADR-style learning records
  reference/          — Cheat sheets and reference docs
  assets/             — Reusable components (stylesheets, quizzes)

Use '--dir <path>' to place the workspace elsewhere.

Examples:
  pharos workspace create "SQL for Research"
  pharos workspace create "Jump Start a Car" --dir ./my-workspace`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		displayName := args[0]
		slug := db.Slugify(displayName)

		// Workspace path — a deployment concern (data dir / --dir override).
		customDir, _ := cmd.Flags().GetString("dir")
		var wsPath string
		if customDir != "" {
			wsPath = customDir
		} else {
			wsPath = filepath.Join(defaultWorkspacesDir(), slug)
		}

		topic, _ := cmd.Flags().GetString("topic")
		if topic == "" {
			topic = displayName
		}

		// CreateWorkspace owns the full row ⇔ dir tree invariant: subdirs,
		// seed templates, and the DB row. The CLI only decides the path.
		created, err := s.CreateWorkspace(slug, topic, wsPath)
		if err != nil {
			return formatError("failed to create workspace", err)
		}

		// Auto-set as current workspace
		_ = s.SetCurrentWorkspace(slug)

		fmt.Println()
		fmt.Printf("  ✓ Created workspace: %s\n", created.DisplayName())
		fmt.Printf("    Path: %s\n", wsPath)
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    cd " + wsPath)
		fmt.Println("    pharos lesson create \"Your first lesson\" --body-file <path>")
		fmt.Println("    pharos record create \"What you learned\" --body-file <path>")
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCreateCmd.Flags().String("dir", "", "Create workspace at a custom path")
	workspaceCreateCmd.Flags().String("topic", "", "Friendly display title for the workspace (default: the name you passed)")
}
