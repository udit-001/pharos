package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var missionCmd = &cobra.Command{
	Use:   "mission",
	Short: "Manage the workspace mission",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Display or edit the MISSION.md file for a workspace.

The mission captures why you're learning a topic and what
success looks like. Every lesson should trace back to it.

Examples:
  pharos mission read --workspace "sql-for-research"
  pharos mission edit --body-file /tmp/mission.md
  pharos mission edit`,
}

var missionReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the workspace mission",
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
	Long: `Update the MISSION.md file from a file.

Examples:
  pharos mission edit --body-file /tmp/mission.md`,
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
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  pharos mission edit --workspace %q --body-file <path>", ws.Name)
		}
		if err := writeWorkspaceFile(wsStore, missionPath, bodyFile, "Mission"); err != nil {
			return err
		}

		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyPageChanged(ws.Name, "mission", 0, "")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(missionCmd)
	missionCmd.AddCommand(missionReadCmd)
	missionCmd.AddCommand(missionEditCmd)
	missionReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	missionEditCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	missionEditCmd.Flags().String("body-file", "", "Read mission content from a file (required)")
}
