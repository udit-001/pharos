package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/udit-001/pharos/internal/vendor"
)

// vendorSyncOpts is the seam tests use to point sync at a temp cache dir and
// an httptest CDN. Production returns the defaults (real CDN, per-OS cache
// dir, 10s budget, stdout progress).
var vendorSyncOpts = func() vendor.SyncOptions {
	return vendor.SyncOptions{Out: os.Stdout}
}

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Manage the global vendored-library cache",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage the global cache of third-party libraries (mermaid, katex,
vega, highlight.js, sql-workbench).

Vendored libraries live in one per-OS cache directory shared by every
workspace — not in each workspace's assets/. The cache is pinned to
versions + sha256 hashes baked into the binary, so every install gets
byte-identical libraries.

'pharos start' runs sync automatically; run it by hand to warm the cache
before going offline.

Examples:
  pharos vendor sync`,
}

var vendorSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fill and verify the global vendor cache",
	Long: `Fetch every pinned vendored library into the global cache,
verifying each file's sha256 against the pin baked into the binary.

Idempotent: a cache that already matches the pin is hash-verified in
place (near-instant) and nothing is downloaded. Offline runs degrade to
skip + log — start still succeeds, features whose library is missing
simply don't render.

Total budget is 10 seconds; whatever doesn't fit is left for the next
start.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := vendorSyncOpts()
		res, err := vendor.Sync(opts)
		if err != nil {
			return err
		}
		if jsonEnabled(cmd) {
			printJSON(map[string]any{
				"fetched":  res.Fetched,
				"verified": res.Verified,
				"skipped":  res.Skipped,
			})
			return nil
		}
		if res.Fetched > 0 {
			fmt.Printf("  ✓ vendored libraries ready (%d fetched, %d verified)\n", res.Fetched, res.Verified)
		} else if res.Skipped == 0 {
			fmt.Printf("  ✓ vendor cache verified (%d files)\n", res.Verified)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vendorCmd)
	vendorCmd.AddCommand(vendorSyncCmd)
}
