package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new question",
	Long: `Create a new question in a workspace.

Questions are DB-only (no file on disk). The --mode flag selects the
config shape and how --body-file is interpreted:

  choice: --body-file is a JSON object {"options": [...], "key": N}
          where "key" is the 0-based index of the correct answer.
  recall: --body-file is the reveal text shown after self-grading.

The slug is derived from the title. Examples:
  pharos question create "What is a JOIN?" --workspace "sql" --mode choice --body-file /tmp/q.json
  pharos question create "Explain MVCC" --workspace "sql" --mode recall --body-file /tmp/reveal.txt`,
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

		mode, _ := cmd.Flags().GetString("mode")
		switch mode {
		case "choice", "recall":
		default:
			return fmt.Errorf("--mode must be \"choice\" or \"recall\", got %q", mode)
		}

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  Write the question content to a file, then: pharos question create %q --workspace %q --mode %q --body-file <path>", title, ws.Name, mode)
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		// Build the typed config from mode + body file, validate it, then
		// marshal back to JSON for storage. All parsing/validation lives
		// behind the QuestionConfig interface.
		var config db.QuestionConfig
		switch mode {
		case "choice":
			var c db.ChoiceConfig
			if err := json.Unmarshal(data, &c); err != nil {
				return formatError("parse choice config", err)
			}
			config = c
		case "recall":
			config = db.RecallConfig{RevealText: string(data)}
		}
		if err := config.Validate(); err != nil {
			return formatError("invalid question config", err)
		}
		configJSON, err := json.Marshal(config)
		if err != nil {
			return formatError("encode question config", err)
		}

		created, err := wsStore.AddQuestion(db.Question{
			Title:  title,
			Mode:   mode,
			Config: string(configJSON),
		})
		if err != nil {
			return formatError("failed to save question", err)
		}

		if jsonOut {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Question created: %s\n", title)
		fmt.Printf("    Slug: %s\n", created.Slug)
		fmt.Printf("    Mode: %s\n", created.Mode)
		fmt.Printf("    Workspace: %s\n", ws.DisplayName())
		fmt.Println()

		return nil
	},
}

func init() {
	questionCmd.AddCommand(questionCreateCmd)
	questionCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	questionCreateCmd.Flags().String("mode", "", "Question mode: choice or recall (required)")
	questionCreateCmd.Flags().String("body-file", "", "Read question content from a file (required)")
}
