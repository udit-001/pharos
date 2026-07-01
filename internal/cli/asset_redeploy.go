package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var assetRedeployCmd = &cobra.Command{
	Use:   "redeploy <name>",
	Short: "Force-sync an asset to the current binary (overwrites)",
	Long: `Re-sync a vendored or seeded asset to the current binary's embedded
files. Re-downloads the lib at its pinned version and overwrites all
companions — use after a binary upgrade or to restore a clobbered file.

For idempotent install (skip if present), use 'pharos asset add' instead.
redeploy overwrites user customizations to a file — consistent across
vendored companions and seeded files.

Examples:
  pharos asset redeploy mermaid
  pharos asset redeploy glossary-tooltip`,
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

		// redeploy always fetches the lib so it's overwritten to the
		// current pinned version.
		libData, err := fetchLib(spec, wsStore, true)
		if err != nil {
			return err
		}

		res, err := wsStore.InstallAsset(spec, libData, true)
		if err != nil {
			return fmt.Errorf("redeploy asset: %w", err)
		}
		_ = wsStore.Touch()

		printInstall(name, spec, res, "Redeployed", wsStore.Workspace())
		return nil
	},
}

func init() {
	assetCmd.AddCommand(assetRedeployCmd)
	assetRedeployCmd.Flags().StringP("workspace", "w", "", "Workspace name")
}
