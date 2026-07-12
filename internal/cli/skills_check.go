package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var skillsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check installed skills and their status",
	Long: `Report which skill locations exist and whether they are current
or outdated.

Checks the Agent Skills Open Standard locations:
  ~/.agents/skills  (global, read by opencode, codex, pi.dev)
  ~/.claude/skills  (global, read by claude-code)
  ./.agents/skills  (project)
  ./.claude/skills  (project)

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
		var outdatedCount int
		for _, loc := range locs {
			if !isSkillInstalled(loc.dir) {
				continue
			}
			installed = true
			icon := "✓"
			if loc.status == "outdated" {
				icon = "⚠"
				outdatedCount++
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
