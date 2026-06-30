package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/urls"
)

var lessonShowCmd = &cobra.Command{
	Use:   "show <seq>",
	Short: "Show a lesson in the dashboard",
	Long: `Print the dashboard URL for a lesson. The dashboard must be running (use 'pharos start').

Examples:
  pharos lesson show 3
  pharos lesson show 3 --workspace "sql-for-research"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		seq, err := parseSeq(args[0])
		if err != nil {
			return err
		}
		wsName, _ := cmd.Flags().GetString("workspace")

		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		// TODO: start dashboard if needed (PID file logic)
		url := "http://127.0.0.1:9090" + urls.Lesson(ws.Name, seq)

		if jsonOut {
			printJSON(map[string]string{"url": url})
			return nil
		}

		fmt.Println()
		fmt.Printf("  View lesson #%d at: %s\n", seq, url)
		fmt.Printf("  Dashboard must be running (use 'pharos start').\n")
		fmt.Println()
		return nil
	},
}

func init() {
	lessonCmd.AddCommand(lessonShowCmd)
	lessonShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
