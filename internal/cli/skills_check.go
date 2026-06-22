package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/skills"
)

var skillsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check installed skills and their status",
	Long: `Report which agents have pharos skills installed and whether
they are current or outdated.

Examples:
  pharos skills check
  pharos skills check --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		type skillStatus struct {
			Agent  string `json:"agent"`
			Scope  string `json:"scope"`
			Skill  string `json:"skill"`
			Status string `json:"status"`
		}

		var results []skillStatus

		for _, a := range agents {
			for _, project := range []bool{false, true} {
				baseDir := a.installDir(project)
				if !isSkillInstalled(baseDir) {
					continue
				}
				scope := "global"
				if project {
					scope = "project"
				}
				for _, skill := range skills.All {
					embedded, err := skillFilesMap(skill)
					if err != nil {
						continue
					}
					status := "current"
					if anySkillChanged(baseDir, skill, embedded) {
						status = "outdated"
					}
					results = append(results, skillStatus{
						Agent:  a.name,
						Scope:  scope,
						Skill:  skill,
						Status: status,
					})
				}
			}
		}

		if jsonOut {
			if results == nil {
				results = []skillStatus{}
			}
			printJSON(results)
			return nil
		}

		fmt.Println()
		if len(results) == 0 {
			fmt.Println("  No skills installed.")
			fmt.Println("  Run 'pharos skills install' to install.")
			fmt.Println()
			return nil
		}

		for _, r := range results {
			statusIcon := "✓"
			if r.Status == "outdated" {
				statusIcon = "⚠"
			}
			fmt.Printf("  %s %s (%s): %s — %s\n", statusIcon, r.Agent, r.Scope, r.Skill, r.Status)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsCheckCmd)
}
