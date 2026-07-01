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
	Short: "Update a quiz's items and/or lesson link",
	Long: `Update a quiz's item list and/or its link to a lesson. At least one of
--items or --lesson must be given; provide both to update together.

--items blocks if the quiz has any in-progress attempts — wait for them to
complete or be abandoned first. --lesson does not block (it is metadata that
doesn't affect a running attempt's questions). Pass --lesson 0 to unlink.

Examples:
  pharos quiz revise sql-basics --items "joins,indexes" --workspace "sql"
  pharos quiz revise sql-basics --lesson 3 --workspace "sql"
  pharos quiz revise sql-basics --items "joins,indexes" --lesson 3 --workspace "sql"
  pharos quiz revise sql-basics --lesson 0 --workspace "sql"   # unlink`,
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

		itemsFlag, _ := cmd.Flags().GetString("items")
		hasItems := strings.TrimSpace(itemsFlag) != ""

		// --lesson 0 means "unlink" (clear the link). parseLessonFlag rejects 0
		// as invalid, so handle the explicit-unlink sentinel here.
		lessonRaw, _ := cmd.Flags().GetString("lesson")
		lessonGiven := strings.TrimSpace(lessonRaw) != ""
		var lessonSeq *int
		if lessonGiven {
			if strings.TrimSpace(lessonRaw) == "0" {
				lessonSeq = nil // explicit unlink
			} else {
				seq, _, perr := parseLessonFlag(cmd)
				if perr != nil {
					return perr
				}
				lessonSeq = &seq
			}
		}

		if !hasItems && !lessonGiven {
			return fmt.Errorf("at least one of --items or --lesson is required\n  pharos quiz revise %q --workspace %q --items \"slug1,slug2\" [--lesson N]", slug, ws.Name)
		}

		var itemCount int
		if hasItems {
			var items []string
			for _, part := range strings.Split(itemsFlag, ",") {
				s := strings.TrimSpace(part)
				if s != "" {
					items = append(items, s)
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
				return formatError("failed to revise quiz items", err)
			}
			itemCount = len(items)
		}

		if lessonGiven {
			lc := wsStore.LessonContent()
			var err error
			if lessonSeq == nil {
				err = lc.ClearQuizLesson(slug)
			} else {
				err = lc.SetQuizLesson(slug, *lessonSeq)
			}
			if err != nil {
				return formatError("failed to revise quiz lesson link", err)
			}
		}

		if jsonOut {
			out := map[string]any{"slug": slug}
			if hasItems {
				out["items"] = itemCount
			}
			if lessonGiven {
				if lessonSeq == nil {
					out["lessonSeq"] = nil
				} else {
					out["lessonSeq"] = *lessonSeq
				}
			}
			printJSON(out)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Quiz revised: %s\n", slug)
		if hasItems {
			fmt.Printf("    Items: %d question(s)\n", itemCount)
		}
		if lessonGiven {
			if lessonSeq == nil {
				fmt.Printf("    Lesson: unlinked\n")
			} else {
				fmt.Printf("    Lesson: #%d\n", *lessonSeq)
			}
		}
		fmt.Println()
		return nil
	},
}

func init() {
	quizCmd.AddCommand(quizReviseCmd)
	quizReviseCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	quizReviseCmd.Flags().String("items", "", "Comma-separated list of question slugs in order")
	quizReviseCmd.Flags().String("lesson", "", "Link to a lesson by sequence number (0 to unlink)")
}
