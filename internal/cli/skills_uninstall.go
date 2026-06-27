package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/skills"
)

var skillsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the pharos skill from an AI agent",
	Long: `Remove pharos skill files that were previously installed.

Reads the pharos.skill.json manifest to determine exactly which
files to delete, then removes them and cleans up empty directories.

Use --agent for non-interactive uninstall, --all to uninstall
from all detected agents, or run without flags for interactive mode.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsUninstall(cmd, args)
	},
}

func runSkillsUninstall(cmd *cobra.Command, args []string) error {
	targets, err := resolveTargets(cmd, "Uninstall", false)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	var errors []string
	for _, t := range targets {
		baseDir := t.agent.installDir(t.project)
		if err := uninstallSkills(baseDir); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", t.agent.name, err))
			continue
		}
		fmt.Printf("  ✓ Uninstalled skills for %s\n", t.agent.name)
	}
	if len(errors) > 0 {
		fmt.Println("  Errors:")
		for _, e := range errors {
			fmt.Printf("    • %s\n", e)
		}
	}
	fmt.Println()
	return nil
}

// uninstallSkills removes all installed skills under baseDir by reading
// each skill's manifest and deleting only the files it lists.
// Falls back to the embedded file list when no manifest exists (pre-upgrade).
func uninstallSkills(baseDir string) error {
	for _, skill := range skills.All {
		skillDir := filepath.Join(baseDir, skill)

		var files []string
		m, err := skills.ReadManifest(skillDir)
		if err == nil {
			files = m.Files
		} else if os.IsNotExist(err) {
			// Pre-upgrade install — no manifest. Fall back to embedded file list.
			embedded, err := skillFilesMap(skill)
			if err != nil {
				return fmt.Errorf("read embedded %s files: %w", skill, err)
			}
			for p := range embedded {
				files = append(files, p)
			}
			sort.Strings(files)
		} else {
			return fmt.Errorf("read manifest for %s: %w", skill, err)
		}

		for _, f := range files {
			p := filepath.Join(skillDir, f)
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				fmt.Printf("  Warning: could not delete %s: %v\n", p, err)
			}
			for parent := filepath.Dir(p); parent != skillDir; parent = filepath.Dir(parent) {
				if err := os.Remove(parent); err != nil {
					break
				}
			}
		}
		if err := os.Remove(skills.ManifestPath(skillDir)); err != nil && !os.IsNotExist(err) {
			fmt.Printf("  Warning: could not delete manifest: %v\n", err)
		}
		os.Remove(skillDir)
	}
	os.Remove(baseDir)
	return nil
}
