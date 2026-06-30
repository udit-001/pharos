package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var quizReadCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Read a quiz's content and metadata",
	Long: `Print a quiz's metadata and its ordered question slugs.

Examples:
  pharos quiz read sql-basics
  pharos quiz read sql-basics --json`,
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

		current, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			return formatError("failed to read quiz", err)
		}
		items, perr := current.ParseItems()
		if perr != nil {
			return formatError("parse quiz items", perr)
		}

		if jsonOut {
			printJSON(map[string]any{
				"id":          current.ID,
				"slug":        current.Slug,
				"title":       current.Title,
				"description": current.Description,
				"items":       items,
				"createdAt":   current.CreatedAt,
				"updatedAt":   current.UpdatedAt,
				"workspace":   ws.Name,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  %s\n", current.Title)
		fmt.Printf("  Slug: %s\n", current.Slug)
		if current.Description != "" {
			fmt.Printf("  Description: %s\n", current.Description)
		}
		fmt.Printf("  Lesson: %s\n", lessonRef(current.LessonSeq))
		fmt.Printf("  Created: %s\n", current.CreatedAt)
		fmt.Printf("  Updated: %s\n", current.UpdatedAt)
		fmt.Printf("  Items: %d question(s)\n", len(items))
		fmt.Println()
		for i, s := range items {
			fmt.Printf("    %d. %s\n", i+1, s)
		}
		if len(items) > 0 {
			fmt.Println()
		}
		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizReadCmd)
	quizReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
