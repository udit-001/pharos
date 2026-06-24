package cli

import (
	"fmt"
	"os"

	"github.com/udit-001/pharos/internal/db"
	"github.com/spf13/cobra"
)

var lessonCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new lesson",
	Long: `Create a new self-contained lesson HTML file in a workspace.

The lesson is created as an HTML file in the workspace's lessons/
directory with sequential numbering and a linked stylesheet.

If no workspace is specified and only one exists, it is used
automatically. If multiple exist, --workspace is required.

Examples:
  pharos lesson create "SQL Joins" --workspace "sql-for-research"
  pharos lesson create "The Connection Sequence" --workspace "jump-start-a-car"
  pharos lesson create "Cadherins" -w "cell-adhesion" --body-file /tmp/lesson.html`,
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
			return fmt.Errorf("--body-file is required\n  Write the lesson HTML to a file, then: pharos lesson create %q --workspace %q --body-file <path>", title, ws.Name)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		created, err := wsStore.CreateLesson(title, string(data))
		if err != nil {
			return formatError("failed to create lesson", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Lesson created: %s\n", title)
		fmt.Printf("    File: %s/%s\n", ws.Path, created.Path)
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

// resolveWorkspace resolves the workspace to use: explicit -w flag, then current
// workspace, then auto-select if only one exists.

func resolveWorkspace(s *db.Store, name string) (*db.WorkspaceStore, error) {
	if name != "" {
		ws, err := s.Workspace(name)
		if err != nil {
			return nil, fmt.Errorf("workspace %q not found\n  Use 'pharos workspace list' to see available workspaces", name)
		}
		return ws, nil
	}

	// Check current workspace
	current, err := s.CurrentWorkspace()
	if err != nil {
		return nil, formatError("failed to get current workspace", err)
	}
	if current != "" {
		ws, err := s.Workspace(current)
		if err == nil {
			return ws, nil
		}
		// Current workspace was deleted — fall through to auto-select
		_ = s.SetCurrentWorkspace("")
	}

	// Auto-select if only one exists
	workspaces, err := s.GetWorkspaces()
	if err != nil {
		return nil, formatError("failed to list workspaces", err)
	}

	switch len(workspaces) {
	case 0:
		return nil, fmt.Errorf("no workspaces found\n  Use 'pharos init' to create one")
	case 1:
		return s.Workspace(workspaces[0].Name)
	default:
		return nil, fmt.Errorf("no current workspace set. You have %d workspaces:\n  Use 'pharos workspace use <name>' to set one, or pass -w to override", len(workspaces))
	}
}

func init() {
	lessonCmd.AddCommand(lessonCreateCmd)
	lessonCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonCreateCmd.Flags().String("body-file", "", "Read lesson HTML content from a file (required)")
}
