package cli

import (
	"fmt"

	"github.com/udit-001/pharos/internal/db"
	"github.com/spf13/cobra"
)

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learning workspaces",
	Long: `List all learning workspaces with stats.

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

		if jsonOut {
			if workspaces == nil {
				workspaces = []db.Workspace{}
			}
			printJSON(workspaces)
			return nil
		}

		fmt.Println()
		if len(workspaces) == 0 {
			fmt.Println("  No workspaces found.")
			fmt.Println("  Use 'pharos init <name>' to create one.")
			fmt.Println()
			return nil
		}

		fmt.Printf("  %d workspace(s)\n\n", len(workspaces))

		rows := make([][]string, 0, len(workspaces))
		for _, w := range workspaces {
			rows = append(rows, []string{
				w.Name,
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
