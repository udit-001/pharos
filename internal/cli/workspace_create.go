package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new learning workspace",
	Long: `Create a new learning workspace.

The workspace is a directory under ~/.pharos/workspaces/ containing:
  MISSION.md          — Why you're learning this topic
  RESOURCES.md        — Curated sources and communities
  GLOSSARY.md         — Canonical terminology
  NOTES.md            — Preferences and working notes
  lessons/            — Self-contained lesson HTML files
  learning-records/   — ADR-style learning records
  reference/          — Cheat sheets and reference docs
  assets/             — Reusable components (stylesheets, quizzes)

Use '--dir <path>' to place the workspace elsewhere, or work
inside the current directory with '--cwd'.

Examples:
  pharos workspace create "SQL for Research"
  pharos workspace create "Jump Start a Car" --dir ./my-workspace
  pharos workspace create "Yoga for Beginners" --cwd`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		displayName := args[0]
		slug := db.Slugify(displayName)

		// Determine workspace path
		useCWD, _ := cmd.Flags().GetBool("cwd")
		customDir, _ := cmd.Flags().GetString("dir")

		var wsPath string
		switch {
		case customDir != "":
			wsPath = customDir
		case useCWD:
			cwd, _ := os.Getwd()
			wsPath = filepath.Join(cwd, slug)
		default:
			wsPath = filepath.Join(defaultWorkspacesDir(), slug)
		}

		// Create workspace directory
		if err := os.MkdirAll(wsPath, 0755); err != nil {
			return fmt.Errorf("create workspace directory: %w", err)
		}

		// Create workspace subdirectories
		for _, d := range []string{"lessons", "learning-records", "reference", "assets"} {
			if err := os.MkdirAll(filepath.Join(wsPath, d), 0755); err != nil {
				return fmt.Errorf("create %s directory: %w", d, err)
			}
		}

		// Create MISSION.md template
		missionFile := filepath.Join(wsPath, "MISSION.md")
		if _, err := os.Stat(missionFile); os.IsNotExist(err) {
			missionContent := fmt.Sprintf(`# Mission: %s

## Why
{1-3 sentences. What changes in your life or work when you have this skill?}

## Success looks like
- {Specific, observable outcome}
- {Another specific outcome}

## Constraints
- {Time, budget, prior commitments}

## Out of scope
- {Adjacent topics you do not want to chase right now}
`, displayName)
			if err := os.WriteFile(missionFile, []byte(missionContent), 0644); err != nil {
				return fmt.Errorf("write MISSION.md: %w", err)
			}
		}

		// Create RESOURCES.md template
		resourcesFile := filepath.Join(wsPath, "RESOURCES.md")
		if _, err := os.Stat(resourcesFile); os.IsNotExist(err) {
			resourcesContent := fmt.Sprintf(`# %s Resources

## Knowledge

- [Resource title](https://example.com)
  What it covers and when to reach for it.

## Wisdom (Communities)

- [Community name](https://example.com)
  What kind of interaction to expect here.

## Gaps
- {Areas where no good resource exists yet}
`, displayName)
			if err := os.WriteFile(resourcesFile, []byte(resourcesContent), 0644); err != nil {
				return fmt.Errorf("write RESOURCES.md: %w", err)
			}
		}

		// Create GLOSSARY.md template
		glossaryFile := filepath.Join(wsPath, "GLOSSARY.md")
		if _, err := os.Stat(glossaryFile); os.IsNotExist(err) {
			glossaryContent := fmt.Sprintf(`# %s Glossary

{One or two sentence description of the topic.}

## Terms

**Term**:
Definition. _Avoid_: Synonyms to avoid.
`, displayName)
			if err := os.WriteFile(glossaryFile, []byte(glossaryContent), 0644); err != nil {
				return fmt.Errorf("write GLOSSARY.md: %w", err)
			}
		}

		// Create NOTES.md
		notesFile := filepath.Join(wsPath, "NOTES.md")
		if _, err := os.Stat(notesFile); os.IsNotExist(err) {
			if err := os.WriteFile(notesFile, []byte("# Notes\n\nPreferences and working notes for this workspace.\n"), 0644); err != nil {
				return fmt.Errorf("write NOTES.md: %w", err)
			}
		}

		// Add to database
		topic, _ := cmd.Flags().GetString("topic")
		if topic == "" {
			topic = displayName
		}
		ws := db.Workspace{
			Name:  slug,
			Topic: topic,
			Path:  wsPath,
		}
		created, err := s.AddWorkspace(ws)
		if err != nil {
			return formatError("failed to create workspace", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Created workspace: %s\n", created.DisplayName())
		fmt.Printf("    Path: %s\n", wsPath)
		// Auto-set as current workspace
		_ = s.SetCurrentWorkspace(slug)

		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    cd " + wsPath)
		fmt.Println("    pharos lesson create \"Your first lesson\"")
		fmt.Println("    pharos record add \"What you learned\"")
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCreateCmd.Flags().Bool("cwd", false, "Create workspace in current directory")
	workspaceCreateCmd.Flags().String("dir", "", "Create workspace at a custom path")
	workspaceCreateCmd.Flags().String("topic", "", "Friendly display title for the workspace (default: the name you passed)")
}
