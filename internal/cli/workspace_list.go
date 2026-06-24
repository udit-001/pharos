package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learning workspaces",
	Long: `List all learning workspaces with stats. The current workspace is marked with *.

Examples:
  pharos workspace list
  pharos workspace list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		workspaces, err := s.GetWorkspaces()
		if err != nil {
			return formatError("failed to list workspaces", err)
		}

		current, _ := s.CurrentWorkspace()

		if jsonOut {
			type wsJSON struct {
				db.Workspace
				Current bool `json:"current"`
			}
			result := make([]wsJSON, len(workspaces))
			for i, w := range workspaces {
				result[i] = wsJSON{Workspace: w, Current: w.Name == current}
			}
			printJSON(result)
			return nil
		}

		fmt.Println()
		if len(workspaces) == 0 {
			fmt.Println("  No workspaces found.")
			fmt.Println("  Use 'pharos init' to create one.")
			fmt.Println()
			return nil
		}

		fmt.Printf("  %d workspace(s)\n\n", len(workspaces))

		rows := make([][]string, 0, len(workspaces))
		for _, w := range workspaces {
			marker := " "
			if w.Name == current {
				marker = "*"
			}
			rows = append(rows, []string{
				fmt.Sprintf("%s %s", marker, w.Name),
				truncate(w.Topic, 30),
				fmt.Sprintf("%d lessons", w.LessonCount),
				fmt.Sprintf("%d records", w.RecordCount),
				formatDateShort(w.LastStudied),
			})
		}

		fmt.Println(formatTable(
			[]string{"Name", "Topic", "Lessons", "Records", "Last Studied"},
			rows,
		))
		fmt.Println()
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
}
