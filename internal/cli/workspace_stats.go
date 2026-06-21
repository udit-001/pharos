package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show learning statistics across all workspaces",
	Long: `Show summary of learning progress across all workspaces.

Examples:
  learn workspace stats
  learn workspace stats --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		workspaces, err := s.GetWorkspaces()
		if err != nil {
			return formatError("failed to get stats", err)
		}

		totalWorkspaces := len(workspaces)
		totalLessons := 0
		totalRecords := 0

		for _, w := range workspaces {
			totalLessons += w.LessonCount
			totalRecords += w.RecordCount
		}

		if jsonOut {
			stats := map[string]any{
				"totalWorkspaces": totalWorkspaces,
				"totalLessons":    totalLessons,
				"totalRecords":    totalRecords,
			}
			printJSON(stats)
			return nil
		}

		fmt.Println()
		fmt.Printf("  Total workspaces:    %d\n", totalWorkspaces)
		fmt.Printf("  Total lessons:       %d\n", totalLessons)
		fmt.Printf("  Total learning recs: %d\n", totalRecords)
		fmt.Println()

		if totalWorkspaces > 0 {
			fmt.Println("  By workspace:")
			for _, w := range workspaces {
				barLessons := ""
				barRecs := ""
				if totalLessons > 0 {
					pct := float64(w.LessonCount) / float64(totalLessons) * 100
					n := int(pct / 5)
					for i := 0; i < n && i < 20; i++ {
						barLessons += "█"
					}
				}
				if totalRecords > 0 {
					pct := float64(w.RecordCount) / float64(totalRecords) * 100
					n := int(pct / 5)
					for i := 0; i < n && i < 20; i++ {
						barRecs += "█"
					}
				}
				fmt.Printf("    %-25s L:%3d %s R:%3d %s\n",
					w.Name, w.LessonCount, barLessons, w.RecordCount, barRecs)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceStatsCmd)
}
