package cli

import (
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agent skills for this project",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Install or uninstall the pharos skill into your AI coding agent so it
knows how to use the CLI to manage learning workspaces.

Installs to the Agent Skills Open Standard locations:
  ~/.agents/skills/  (global, read by opencode, codex, pi.dev)
  ~/.claude/skills/  (global, read by claude-code)

Use --project to install at the project level instead.

Run 'pharos skills check' to see all installed copies and their status.`,
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the pharos skill",
	Long: `Install the pharos skill for your AI coding agent.

Installs to the Agent Skills Open Standard location (~/.agents/skills/)
which is read by opencode, codex, and pi.dev. Also installs to
~/.claude/skills/ for claude-code if detected.

Flags:
  --all          Install all detected families (default in non-interactive mode)
  --agents-only  Install only to ~/.agents/skills (opencode, codex, pi.dev)
  --claude-only  Install only to ~/.claude/skills (claude-code)
  --project      Install at project level (./.agents/skills) instead of global

Run without flags for interactive mode.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsInstall(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	skillsCmd.AddCommand(skillsUninstallCmd)
	skillsInstallCmd.Flags().Bool("agents-only", false, "Install only to .agents/skills (opencode, codex, pi.dev)")
	skillsInstallCmd.Flags().Bool("claude-only", false, "Install only to .claude/skills (claude-code)")
	skillsInstallCmd.Flags().Bool("all", false, "Install all detected families")
	skillsInstallCmd.Flags().Bool("project", false, "Install at project level instead of globally")
	skillsUninstallCmd.Flags().Bool("all", false, "Remove all discovered installs")
}
