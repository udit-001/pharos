package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
  learn lesson create "SQL Joins" --workspace "sql-for-research"
  learn lesson create "The Connection Sequence" --workspace "jump-start-a-car"
  learn lesson create "Cadherins" -w "cell-adhesion" --open`,
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

		// Determine next sequence number
		lessons, err := wsStore.GetLessons()
		if err != nil {
			return formatError("failed to get lessons", err)
		}
		seqNum := len(lessons) + 1

		// Create filename
		slug := slugify(title)
		filename := fmt.Sprintf("%04d-%s.html", seqNum, slug)
		lessonPath := filepath.Join(ws.Path, "lessons", filename)

		// Lesson content comes from --body-file (required) — no stub template.
		// The teach skill writes the HTML to a temp file and passes it here, so
		// multiline content lands verbatim without shell-escaping.
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  Write the lesson HTML to a file, then: learn lesson create %q --workspace %q --body-file <path>", title, ws.Name)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}
		html := string(data)

		if err := os.WriteFile(lessonPath, []byte(html), 0644); err != nil {
			return fmt.Errorf("write lesson file: %w", err)
		}

		// Save to database
		created, err := wsStore.AddLesson(db.Lesson{
			Title:    title,
			Filename: filename,
			Path:     filepath.Join("lessons", filename),
		})
		if err != nil {
			return formatError("failed to save lesson", err)
		}

		_ = wsStore.Touch()

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Lesson created: %s\n", title)
		fmt.Printf("    File: %s\n", lessonPath)
		fmt.Printf("    Workspace: %s\n", ws.Name)
		fmt.Println()

		openFile, _ := cmd.Flags().GetBool("open")
		if openFile {
			fmt.Printf("  Opening %s ...\n", filename)
			openDirInExplorer(lessonPath)
		}

		return nil
	},
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric except hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func resolveWorkspace(s *db.Store, name string) (*db.WorkspaceStore, error) {
	if name != "" {
		ws, err := s.Workspace(name)
		if err != nil {
			return nil, fmt.Errorf("workspace %q not found\n  Use 'learn workspace list' to see available workspaces", name)
		}
		return ws, nil
	}

	// No name given — auto-select if only one exists
	workspaces, err := s.GetWorkspaces()
	if err != nil {
		return nil, formatError("failed to list workspaces", err)
	}

	switch len(workspaces) {
	case 0:
		return nil, fmt.Errorf("no workspaces found\n  Use 'learn init <name>' to create one")
	case 1:
		return s.Workspace(workspaces[0].Name)
	default:
		return nil, fmt.Errorf("multiple workspaces found — use --workspace to specify one")
	}
}

func init() {
	lessonCmd.AddCommand(lessonCreateCmd)
	lessonCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonCreateCmd.Flags().Bool("open", false, "Open the lesson file after creation")
	lessonCreateCmd.Flags().String("body-file", "", "Read lesson HTML content from a file (required)")
}
