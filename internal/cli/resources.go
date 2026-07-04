package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Manage the workspace resources",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Display or edit the RESOURCES.md file for a workspace.

Resources are the curated set of trusted sources for this topic.
Knowledge comes from high-quality resources listed here.

Examples:
  pharos resources read --workspace "sql-for-research"
  pharos resources edit --body-file /tmp/resources.md
  pharos resources edit`,
}

var resourcesReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read the workspace resources",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		resourcePath := filepath.Join(ws.Path, "RESOURCES.md")
		if jsonOut {
			return readAndPrintJSON(ws, resourcePath, "RESOURCES.md")
		}
		return readAndPrintFile(ws, resourcePath)
	},
}

var resourcesEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the workspace resources",
	Long: `Update the RESOURCES.md file from a file.

Examples:
  pharos resources edit --body-file /tmp/resources.md`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()
		resourcesPath := filepath.Join(ws.Path, "RESOURCES.md")

		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  pharos resources edit --workspace %q --body-file <path>", ws.Name)
		}
		if err := writeWorkspaceFile(wsStore, resourcesPath, bodyFile, "Resources"); err != nil {
			return err
		}

		notifyServer("workspace:"+ws.Name, "changed", 0)
		notifyPageChanged(ws.Name, "resources", 0, "")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resourcesCmd)
	resourcesCmd.AddCommand(resourcesReadCmd)
	resourcesCmd.AddCommand(resourcesEditCmd)
	resourcesReadCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	resourcesEditCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	resourcesEditCmd.Flags().String("body-file", "", "Read resources content from a file (required)")
}
