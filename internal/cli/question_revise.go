package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var questionReviseCmd = &cobra.Command{
	Use:   "revise <slug>",
	Short: "Update a question's title, mode, and/or config",
	Long: `Update a question in place. The slug stays stable — it is not
regenerated from the title, so existing quiz item references remain valid.

At least one of --title, --mode, --body-file, --stimulus-file, or
--clear-stimulus is required. When --mode is given, --body-file is required
(the config shape changes with mode). When --body-file is given without
--mode, the config is parsed per the question's current mode. --stimulus-file
and --clear-stimulus are mutually exclusive.

Examples:
  pharos question revise sql-join-q --title "What is a LEFT JOIN?"
  pharos question revise sql-join-q --body-file /tmp/fixed.json
  pharos question revise sql-join-q --mode recall --body-file /tmp/reveal.txt
  pharos question revise sql-join-q --stimulus-file /tmp/chart.html
  pharos question revise sql-join-q --clear-stimulus`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		titleFlag, _ := cmd.Flags().GetString("title")
		modeFlag, _ := cmd.Flags().GetString("mode")
		bodyFile, _ := cmd.Flags().GetString("body-file")
		stimulusFile, _ := cmd.Flags().GetString("stimulus-file")
		clearStimulus, _ := cmd.Flags().GetBool("clear-stimulus")

		hasTitle := titleFlag != ""
		hasMode := modeFlag != ""
		hasBody := bodyFile != ""
		hasStimulus := stimulusFile != ""
		hasClear := clearStimulus

		if !hasTitle && !hasMode && !hasBody && !hasStimulus && !hasClear {
			return fmt.Errorf("at least one of --title, --mode, --body-file, --stimulus-file, or --clear-stimulus is required\n  pharos question revise %q --workspace %q --title \"New title\" [--mode choice|recall --body-file <path>] [--stimulus-file <path> | --clear-stimulus]", slug, wsName)
		}
		if hasMode && !hasBody {
			return fmt.Errorf("--mode requires --body-file (the config shape changes with mode)")
		}
		if hasStimulus && hasClear {
			return fmt.Errorf("--stimulus-file and --clear-stimulus are mutually exclusive")
		}

		var titlePtr *string
		if hasTitle {
			titlePtr = &titleFlag
		}
		var modePtr *string
		if hasMode {
			switch modeFlag {
			case "choice", "recall":
			default:
				return fmt.Errorf("--mode must be \"choice\" or \"recall\", got %q", modeFlag)
			}
			modePtr = &modeFlag
		}

		var configPtr *string
		if hasBody {
			data, err := readBodyFile(bodyFile)
			if err != nil {
				return err
			}

			effectiveMode := modeFlag
			if effectiveMode == "" {
				current, err := wsStore.GetQuestionBySlug(slug)
				if err != nil {
					return formatError("find question", err)
				}
				effectiveMode = current.Mode
			}

			var config db.QuestionConfig
			switch effectiveMode {
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
			cfg := string(configJSON)
			configPtr = &cfg
		}

		var stimulusPtr *string
		if hasStimulus {
			data, err := readBodyFile(stimulusFile)
			if err != nil {
				return err
			}
			s := string(data)
			stimulusPtr = &s
		} else if hasClear {
			s := ""
			stimulusPtr = &s
		}

		updated, err := wsStore.ReviseQuestion(slug, titlePtr, modePtr, configPtr, stimulusPtr)
		if err != nil {
			return formatError("failed to revise question", err)
		}

		ws := wsStore.Workspace()
		notifyServer("workspace:"+ws.Name, "changed", 0)

		if jsonOut {
			printJSON(updated)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Question revised: %s\n", slug)
		if hasTitle {
			fmt.Printf("    Title: %s\n", titleFlag)
		}
		if hasMode {
			fmt.Printf("    Mode: %s\n", modeFlag)
		}
		if hasBody {
			fmt.Printf("    Config: updated\n")
		}
		if hasStimulus {
			fmt.Printf("    Stimulus: updated\n")
		}
		if hasClear {
			fmt.Printf("    Stimulus: cleared\n")
		}
		fmt.Println()
		return nil
	},
}

func init() {
	questionCmd.AddCommand(questionReviseCmd)
	questionReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	questionReviseCmd.Flags().String("title", "", "Update the question title")
	questionReviseCmd.Flags().String("mode", "", "Update the question mode: choice or recall (requires --body-file)")
	questionReviseCmd.Flags().String("body-file", "", "Read new question config from a file")
	questionReviseCmd.Flags().String("stimulus-file", "", "Read a stimulus HTML file (replaces the existing stimulus)")
	questionReviseCmd.Flags().Bool("clear-stimulus", false, "Remove the question's stimulus")
}
