package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new question",
	Long: `Create a new question in a workspace.

The --mode flag selects the config shape and how --body-file is
interpreted. A question may optionally carry a stimulus (--stimulus-file),
a standalone HTML file rendered in an iframe above the answer:

  choice: --body-file is a JSON object {"options": [...], "key": N}
          where "key" is the 0-based index of the correct answer.
  recall: --body-file is the reveal text shown after self-grading.

The slug is derived from the title. Examples:
  pharos question create "What is a JOIN?" --workspace "sql" --mode choice --body-file /tmp/q.json
  pharos question create "Which segment grew most?" --workspace "sql" --mode choice --body-file /tmp/q.json --stimulus-file /tmp/chart.html
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
		data, err := readBodyFile(bodyFile)
		if err != nil {
			return err
		}

		stimulusFile, _ := cmd.Flags().GetString("stimulus-file")
		stimulusHTML := ""
		if stimulusFile != "" {
			sdata, err := readBodyFile(stimulusFile)
			if err != nil {
				return err
			}
			stimulusHTML = string(sdata)
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
		}, stimulusHTML)
		if err != nil {
			return formatError("failed to save question", err)
		}

		notifyServer("workspace:"+ws.Name, "changed", 0)

		if jsonEnabled(cmd) {
			printJSON(created)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Question created: %s\n", title)
		fmt.Printf("    Slug: %s\n", created.Slug)
		fmt.Printf("    Mode: %s\n", created.Mode)
		if created.Filename != "" {
			fmt.Printf("    Stimulus: %s\n", created.Filename)
		}
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
	questionCreateCmd.Flags().String("stimulus-file", "", "Read a stimulus HTML file (optional; rendered in an iframe above the answer)")
}
