package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var skillsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check installed skills and their status",
	Long: `Report which skill locations exist and whether they are current,
outdated, or orphaned (installed at a location pharos no longer manages).

Scans every directory each provider reads — global, project, and
ancestor — so stale copies that shadow fresh installs are surfaced.

Examples:
  pharos skills check
  pharos skills check --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		locs := discover()

		if jsonOut {
			type skillStatus struct {
				Dir       string   `json:"dir"`
				Scope     string   `json:"scope"`
				Family    string   `json:"family"`
				Status    string   `json:"status"`
				Readers   []string `json:"readers"`
				Unmanaged bool     `json:"unmanaged"`
			}
			var results []skillStatus
			for _, loc := range locs {
				if !isSkillInstalled(loc.dir) {
					continue
				}
				results = append(results, skillStatus{
					Dir:       loc.dir,
					Scope:     loc.scope,
					Family:    loc.family,
					Status:    loc.status,
					Readers:   loc.readers,
					Unmanaged: loc.unmanaged,
				})
			}
			if results == nil {
				results = []skillStatus{}
			}
			printJSON(results)
			return nil
		}

		fmt.Println()
		var installed bool
		var orphanCount, outdatedCount int
		for _, loc := range locs {
			if !isSkillInstalled(loc.dir) {
				continue
			}
			installed = true
			icon := "✓"
			switch loc.status {
			case "outdated":
				icon = "⚠"
				outdatedCount++
			case "orphan":
				icon = "⚠"
				orphanCount++
			}
			fmt.Printf("  %s %s\n", icon, formatLocationLine(loc))
		}

		if !installed {
			fmt.Println("  No skills installed.")
			fmt.Println("  Run 'pharos skills install' to install.")
			fmt.Println()
			return nil
		}

		fmt.Println()
		if orphanCount > 0 {
			fmt.Printf("  %d orphaned install(s) found. Run 'pharos skills uninstall --orphans' to remove them.\n", orphanCount)
		}
		if outdatedCount > 0 {
			fmt.Printf("  %d outdated install(s) found. Run 'pharos skills install' to update.\n", outdatedCount)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsCheckCmd)
}
