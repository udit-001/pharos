package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var lessonMoveCmd = &cobra.Command{
	Use:   "move <slug>",
	Short: "Change a lesson's position in the sidebar",
	Long: `Move a lesson to a new position in the learning path.

The lesson is identified by its slug. Use --before or --after to place it
relative to another lesson, or --first/--last for the edges.

Examples:
  pharos lesson move sql-joins --before where-clauses
  pharos lesson move sql-joins --after where-clauses
  pharos lesson move intro --first
  pharos lesson move advanced --last`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		fromSlug := args[0]

		// Determine target position.
		before, _ := cmd.Flags().GetString("before")
		after, _ := cmd.Flags().GetString("after")
		first, _ := cmd.Flags().GetBool("first")
		last, _ := cmd.Flags().GetBool("last")

		var pos, targetSlug string
		switch {
		case first:
			pos = "first"
		case last:
			pos = "last"
		case before != "":
			pos = "before"
			targetSlug = before
		case after != "":
			pos = "after"
			targetSlug = after
		default:
			return fmt.Errorf("specify a position: --before, --after, --first, or --last")
		}

		oldSeq, newSeq, err := wsStore.MoveLesson(fromSlug, targetSlug, pos)
		if err != nil {
			return formatError("move lesson", err)
		}

		// Live-sync: refresh sidebar + navigate if viewing the moved lesson.
		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyServerFull("workspace:"+ws.Name, "navigate", "", 0,
			fmt.Sprintf("/workspace/%s/lesson/%s", ws.Name, fromSlug))

		if jsonEnabled(cmd) {
			printJSON(map[string]any{
				"slug":   fromSlug,
				"oldSeq": oldSeq,
				"newSeq": newSeq,
				"url":    fmt.Sprintf("/workspace/%s/lesson/%s", ws.Name, fromSlug),
			})
			return nil
		}

		// Fetch sidebar for snapshot.
		sd, _ := wsStore.GetSidebarData()
		var names []string
		for _, l := range sd.Lessons {
			names = append(names, l.Title)
		}

		fmt.Println()
		fmt.Printf("  ✓ Moved %q from #%d to #%d\n", fromSlug, oldSeq, newSeq)
		if len(names) > 0 {
			fmt.Printf("    Sidebar: %s\n", strings.Join(names, " → "))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	lessonCmd.AddCommand(lessonMoveCmd)
	lessonMoveCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	lessonMoveCmd.Flags().String("before", "", "Place before this lesson (by slug)")
	lessonMoveCmd.Flags().String("after", "", "Place after this lesson (by slug)")
	lessonMoveCmd.Flags().Bool("first", false, "Move to the first position")
	lessonMoveCmd.Flags().Bool("last", false, "Move to the last position")
}
