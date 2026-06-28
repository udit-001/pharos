package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new quiz",
	Long: `Create a new quiz from an ordered list of question slugs.

The --items flag is a comma-separated list of question slugs in the
order they should be presented. The slug is derived from the title.

Examples:
  pharos quiz create "SQL Basics" --items "what-is-a-join,explain-mvcc" --workspace "sql"
  pharos quiz create "Genetics" --items "chd8-gene,heritability" --workspace "autism" --description "Foundations"`,
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

		itemsFlag, _ := cmd.Flags().GetString("items")
		if strings.TrimSpace(itemsFlag) == "" {
			return fmt.Errorf("--items is required\n  List question slugs separated by commas, then: pharos quiz create %q --workspace %q --items \"slug1,slug2\"", title, ws.Name)
		}
		description, _ := cmd.Flags().GetString("description")

		var items []string
		for _, part := range strings.Split(itemsFlag, ",") {
			slug := strings.TrimSpace(part)
			if slug != "" {
				items = append(items, slug)
			}
		}
		if len(items) == 0 {
			return fmt.Errorf("--items must list at least one question slug")
		}
		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return formatError("encode quiz items", err)
		}

		created, err := wsStore.AddQuiz(db.Quiz{
			Title:       title,
			Description: description,
			Items:       string(itemsJSON),
		})
		if err != nil {
			return formatError("failed to save quiz", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Quiz created: %s\n", title)
		fmt.Printf("    Slug: %s\n", created.Slug)
		fmt.Printf("    Items: %d question(s)\n", len(items))
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizCreateCmd)
	quizCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	quizCreateCmd.Flags().String("items", "", "Comma-separated list of question slugs in order (required)")
	quizCreateCmd.Flags().String("description", "", "Short description of the quiz")
}
