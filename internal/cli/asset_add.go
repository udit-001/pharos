package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var assetAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Install a vendored or seeded asset (if absent)",
	Long: `Install a known vendored or seeded asset into the workspace's
assets/ directory. Skips files already present — idempotent. Use
'pharos asset redeploy' to force-sync to the current binary.

Assets ship at their pinned, tested version — no @version override (the
embedded companions are authored for that version only).

Use 'pharos asset list' to see available assets and which are present.

Examples:
  pharos asset add mermaid
  pharos asset add highlightjs
  pharos asset add glossary-tooltip`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}

		name := args[0]
		spec, ok := knownAssets[name]
		if !ok {
			return fmt.Errorf("unknown asset %q\n  Available: %s", name, knownAssetsString())
		}

		// Idempotent add: don't re-download a lib that's already present.
		libData, err := fetchLib(spec, wsStore, false)
		if err != nil {
			return err
		}

		res, err := wsStore.InstallAsset(spec, libData, false)
		if err != nil {
			return fmt.Errorf("add asset: %w", err)
		}
		_ = wsStore.Touch()

		printInstall(name, spec, res, "Added", wsStore.Workspace())
		return nil
	},
}

func init() {
	assetCmd.AddCommand(assetAddCmd)
	assetAddCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
