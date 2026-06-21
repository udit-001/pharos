package cli

import (
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agent skills for this project",
	Long: `Install the learn skill into your AI coding agent so it
knows how to use the CLI to manage learning workspaces.

Supports: opencode, claude-code, codex, pi.dev`,
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the learn skill into an AI agent",
	Long: `Interactively install the learn skill for your AI coding agent.
The skill teaches the agent how to use the learn CLI commands.

Supported agents:
  opencode     Installs to .opencode/skills/learn/
  claude-code  Installs to .claude/skills/learn/
  codex        Installs to .codex/skills/learn/
  pi.dev       Installs to .pi/skills/learn/

Installs the full skill (SKILL.md + references/).

Run without flags for interactive mode, or pass --agent to skip prompts.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsInstall(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	skillsInstallCmd.Flags().String("agent", "", "Agent to install for (opencode, claude-code, codex, pi)")
}
