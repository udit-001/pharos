package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/skills"
)

var skillsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the pharos skill",
	Long: `Remove pharos skill files that were previously installed.

Scans the standard install locations (global + project) for installed
copies and offers interactive removal.

Examples:
  pharos skills uninstall
  pharos skills uninstall --all`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsUninstall(cmd, args)
	},
}

func runSkillsUninstall(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")

	locs := discover()

	var installed []skillLocation
	for _, loc := range locs {
		if isSkillInstalled(loc.dir) {
			installed = append(installed, loc)
		}
	}

	if len(installed) == 0 {
		fmt.Println()
		fmt.Println("  No skills installed.")
		fmt.Println()
		return nil
	}

	if all {
		return uninstallAll(installed)
	}

	return uninstallInteractive(installed)
}

func uninstallAll(installed []skillLocation) error {
	fmt.Println()
	fmt.Printf("  Found %d skill install(s):\n\n", len(installed))
	for _, loc := range installed {
		fmt.Printf("    %s\n", formatLocationLine(loc))
	}
	fmt.Printf("\n  Remove all %d? [y/N] ", len(installed))
	if !promptYes() {
		fmt.Println("  Cancelled.")
		return nil
	}
	fmt.Println()
	var errors []string
	for _, loc := range installed {
		if err := uninstallSkills(loc.dir); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", loc.dir, err))
			continue
		}
		fmt.Printf("  ✓ Removed %s\n", loc.dir)
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

func uninstallInteractive(installed []skillLocation) error {
	fmt.Println()
	fmt.Printf("  Found %d skill install(s):\n\n", len(installed))
	for i, loc := range installed {
		fmt.Printf("    %d. %s\n", i+1, formatLocationLine(loc))
	}
	fmt.Println()
	fmt.Print("  Remove which? (comma-separated numbers, 'all', or 0 to cancel)\n  > ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "0" {
		fmt.Println("  Cancelled.")
		return nil
	}
	if input == "all" {
		return uninstallAll(installed)
	}

	// Parse comma-separated numbers.
	var selected []skillLocation
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(installed) {
			fmt.Printf("  Ignoring invalid input: %s\n", part)
			continue
		}
		selected = append(selected, installed[n-1])
	}
	if len(selected) == 0 {
		fmt.Println("  Nothing selected.")
		return nil
	}
	fmt.Println()
	var errors []string
	for _, loc := range selected {
		if err := uninstallSkills(loc.dir); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", loc.dir, err))
			continue
		}
		fmt.Printf("  ✓ Removed %s\n", loc.dir)
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
		if _, err := os.Stat(skills.ManifestPath(skillDir)); err != nil {
			embedded, err := skillFilesMap(skill)
			if err != nil {
				return fmt.Errorf("read embedded %s files: %w", skill, err)
			}
			for p := range embedded {
				files = append(files, p)
			}
			sort.Strings(files)
		} else {
			m, err := skills.ReadManifest(skillDir)
			if err != nil {
				return fmt.Errorf("read manifest for %s: %w", skill, err)
			}
			files = m.Files
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
