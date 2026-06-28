package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var quizReviseCmd = &cobra.Command{
	Use:   "revise <slug>",
	Short: "Update a quiz's question items",
	Long: `Update a quiz's item list with a new set of question slugs.

Blocks if the quiz has any in-progress attempts — the agent must wait for
them to complete or be abandoned first.

Examples:
  pharos quiz revise genetics-foundations --items "q1,q2,q3" --workspace "autism"
  pharos quiz revise sql-basics --items "joins,indexes" --workspace "sql"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		itemsFlag, _ := cmd.Flags().GetString("items")
		if strings.TrimSpace(itemsFlag) == "" {
			return fmt.Errorf("--items is required\n  pharos quiz revise %q --workspace %q --items \"slug1,slug2\"", slug, wsStore.Workspace().Name)
		}

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

		if err := wsStore.UpdateQuizItems(slug, string(itemsJSON)); err != nil {
			if errors.Is(err, db.ErrQuizHasInProgress) {
				return fmt.Errorf("quiz %q has in-progress attempts\n  Wait for them to complete, or abandon them first", slug)
			}
			return formatError("failed to revise quiz", err)
		}

		if jsonOut {
			printJSON(map[string]any{"slug": slug, "items": items})
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Quiz revised: %s\n", slug)
		fmt.Printf("    Items: %d question(s)\n", len(items))
		fmt.Println()
		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizReviseCmd)
	quizReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	quizReviseCmd.Flags().String("items", "", "Comma-separated list of question slugs in order (required)")
}
