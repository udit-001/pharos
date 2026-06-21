package cli

import (
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agent skills for this project",
	Long: `Install the pharos skill into your AI coding agent so it
knows how to use the CLI to manage learning workspaces.

Installs globally by default (~/.agent/skills/). Use --project
to install at the project level (./.agent/skills/) instead.

Supported: opencode, claude-code, codex, pi.dev`,
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the pharos skill into an AI agent",
	Long: `Interactively install the pharos skill for your AI coding agent.
The skill teaches the agent how to use the pharos CLI commands.

Supported agents:
  opencode     Installs to ~/.opencode/skills/ (global) or ./.opencode/skills/ (--project)
  claude-code  Installs to ~/.claude/skills/ (global) or ./.claude/skills/ (--project)
  codex        Installs to ~/.codex/skills/ (global) or ./.codex/skills/ (--project)
  pi.dev       Installs to ~/.pi/skills/ (global) or ./.pi/skills/ (--project)

Default is global install (home directory). Interactive mode prompts
for the install location. Use --project with --agent for
non-interactive project-level install.

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
	skillsInstallCmd.Flags().Bool("project", false, "Install at project level instead of globally")
}
