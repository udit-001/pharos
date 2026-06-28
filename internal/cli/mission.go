package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Manage the workspace mission",
	Long: `Display or edit the MISSION.md file for a workspace.

The mission captures why you're learning a topic and what
success looks like. Every lesson should trace back to it.

Examples:
  pharos mission show --workspace "sql-for-research"
  pharos mission edit --body-file /tmp/mission.md
  pharos mission edit`,
}

var missionShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the workspace mission",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		missionPath := filepath.Join(ws.Path, "MISSION.md")
		if jsonOut {
			return readAndPrintJSON(ws, missionPath, "MISSION.md")
		}
		return readAndPrintFile(ws, missionPath)
	},
}

var missionEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the workspace mission",
	Long: `Update the MISSION.md file from a file or open in $EDITOR.

Examples:
  pharos mission edit --body-file /tmp/mission.md
  pharos mission edit`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		missionPath := filepath.Join(ws.Path, "MISSION.md")

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile != "" {
			return writeWorkspaceFile(wsStore, missionPath, bodyFile, "Mission")
		}

		return openInEditor(wsStore, missionPath, "MISSION.md")
	},
}

func init() {
	rootCmd.AddCommand(missionCmd)
	missionCmd.AddCommand(missionShowCmd)
	missionCmd.AddCommand(missionEditCmd)
	missionShowCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	missionEditCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	missionEditCmd.Flags().String("body-file", "", "Write content from a file (non-interactive)")
}
