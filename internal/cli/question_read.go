package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionReadCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Read a question's content and metadata",
	Long: `Print a question's metadata and config. For choice mode, the correct
option is marked. For recall mode, the reveal text is shown.

Examples:
  pharos question read what-is-a-join
  pharos question read what-is-a-join --json`,
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

		current, err := wsStore.GetQuestionBySlug(slug)
		if err != nil {
			return formatError("failed to read question", err)
		}
		config, perr := current.ParseConfig()
		if perr != nil {
			return formatError("parse question config", perr)
		}

		if jsonOut {
			printJSON(map[string]any{
				"id":        current.ID,
				"slug":      current.Slug,
				"title":     current.Title,
				"mode":      current.Mode,
				"createdAt": current.CreatedAt,
				"updatedAt": current.UpdatedAt,
				"workspace": ws.Name,
				"config":    config,
			})
			return nil
		}

		fmt.Println()
		fmt.Printf("  %s\n", current.Title)
		fmt.Printf("  Slug: %s\n", current.Slug)
		fmt.Printf("  Mode: %s\n", current.Mode)
		fmt.Printf("  Created: %s\n", current.CreatedAt)
		fmt.Printf("  Updated: %s\n", current.UpdatedAt)
		fmt.Println()

		switch c := config.(type) {
		case db.ChoiceConfig:
			fmt.Println("  Options:")
			for i, opt := range c.Options {
				mark := " "
				if i == c.Key {
					mark = "✓"
				}
				letter := string(rune('A' + i))
				fmt.Printf("    (%s) %s   %s\n", letter, opt, mark)
			}
		case db.RecallConfig:
			fmt.Println("  Reveal:")
			for _, line := range strings.Split(c.RevealText, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
		fmt.Println()
		return nil
	},
}

func init() {
	questionCmd.AddCommand(questionReadCmd)
	questionReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
