package cli

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/skills"
)

type agentTarget struct {
	name    string
	subdir  string // relative path under the install root, e.g. ".opencode/skills"
	aliases []string
	detect  func() bool
}

func (a agentTarget) globalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return a.localDir()
	}
	return filepath.Join(home, a.subdir)
}

func (a agentTarget) localDir() string {
	return a.subdir
}

func (a agentTarget) installDir(project bool) string {
	if project {
		return a.localDir()
	}
	return a.globalDir()
}

var agents = []agentTarget{
	{name: "opencode", subdir: ".opencode/skills", detect: func() bool { return hasBinary("opencode") || hasDir(".opencode") }},
	{name: "claude-code", subdir: ".claude/skills", detect: func() bool { return hasBinary("claude") || hasDir(".claude") }},
	{name: "codex", subdir: ".codex/skills", detect: func() bool { return hasBinary("codex") || hasDir(".codex") }},
	{name: "pi.dev", subdir: ".pi/skills", aliases: []string{"pi"}, detect: func() bool { return hasBinary("pi") || hasDir(".pi") }},
}

type installTarget struct {
	agent   agentTarget
	project bool
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	targets, err := resolveTargets(cmd, "Install", true)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	var errors []string
	for _, t := range targets {
		baseDir := t.agent.installDir(t.project)
		n, err := installAllSkills(baseDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", t.agent.name, err))
			continue
		}
		fmt.Printf("  ✓ Installed %d skill(s) to %s/ (%d files)\n", len(skills.All), baseDir, n)
	}
	fmt.Println()
	printNextSteps()
	if len(errors) > 0 {
		fmt.Println("  Errors:")
		for _, e := range errors {
			fmt.Printf("    • %s\n", e)
		}
	}
	fmt.Println()
	return nil
}

// resolveTargets resolves --agent, --all, and interactive modes into a list
// of install targets. Callers provide a verb ("Install" or "Uninstall") for
// prompt text and allDefault for the --all prompt's default answer.
func resolveTargets(cmd *cobra.Command, verb string, allDefault bool) ([]installTarget, error) {
	agent, _ := cmd.Flags().GetString("agent")
	all, _ := cmd.Flags().GetBool("all")

	if agent != "" && all {
		return nil, fmt.Errorf("--agent and --all are mutually exclusive")
	}

	var project bool
	if all {
		project, _ = cmd.Flags().GetBool("project")
		detected := detectAgents()
		if len(detected) == 0 {
			fmt.Println("  No AI coding agents detected.")
			return nil, nil
		}
		fmt.Printf("  %s pharos skills for all detected agents? ", verb)
		if allDefault {
			fmt.Print("[Y/n] ")
			if !promptDefaultYes() {
				fmt.Println("  Skipped.")
				return nil, nil
			}
		} else {
			fmt.Print("[y/N] ")
			if !promptYes() {
				fmt.Println("  Skipped.")
				return nil, nil
			}
		}
		targets := make([]installTarget, len(detected))
		for i, a := range detected {
			targets[i] = installTarget{a, project}
		}
		return targets, nil
	}

	if agent != "" {
		project, _ = cmd.Flags().GetBool("project")
		selected, err := resolveAgent(agent)
		if err != nil {
			return nil, err
		}
		return []installTarget{{selected, project}}, nil
	}

	selected := pickAgent()
	project = promptScope()
	return []installTarget{{selected, project}}, nil
}

func resolveAgent(name string) (agentTarget, error) {
	for _, a := range agents {
		if a.name == name {
			return a, nil
		}
		for _, alias := range a.aliases {
			if alias == name {
				return a, nil
			}
		}
	}
	var supported []string
	for _, a := range agents {
		supported = append(supported, a.name)
	}
	return agentTarget{}, fmt.Errorf("unknown agent %q\n  Supported: %s\n  If %q is a new AI coding agent, open an issue at https://github.com/udit-001/pharos/issues/new", name, strings.Join(supported, ", "), name)
}

// promptScope asks the user whether to install globally or at project level.
func promptScope() bool {
	fmt.Println()
	fmt.Println("  Install location:")
	fmt.Println("    1. Globally — available to all projects (~/.agent/skills/)")
	fmt.Println("    2. This project — only in current directory (./.agent/skills/)")
	fmt.Println()
	fmt.Print("  Enter number [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "2" {
		return true // project-level
	}
	return false // global (default)
}

// installAllSkills installs every embedded skill (teach, ...) into
// baseDir/<skillName>, then writes a manifest for change detection.
// Returns the total file count across all skills.
func installAllSkills(baseDir string) (int, error) {
	total := 0
	for _, skill := range skills.All {
		skillDir := filepath.Join(baseDir, skill)
		files, err := skillFilesMap(skill)
		if err != nil {
			return total, fmt.Errorf("read %s skill files: %w", skill, err)
		}
		n, err := writeSkillFiles(files, skillDir)
		if err != nil {
			return total, fmt.Errorf("install %s skill: %w", skill, err)
		}
		total += n

		// Write manifest for change detection and uninstall
		hash := skills.ManifestHash(files)
		relPaths := make([]string, 0, len(files))
		for p := range files {
			relPaths = append(relPaths, p)
		}
		sort.Strings(relPaths)
		if err := skills.WriteManifest(skillDir, relPaths, hash); err != nil {
			return total, fmt.Errorf("write manifest for %s: %w", skill, err)
		}
	}
	return total, nil
}

// writeSkillFiles writes every entry in files under destDir.
func writeSkillFiles(files map[string][]byte, destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create directories: %w", err)
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		out := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return 0, err
		}
		if err := os.WriteFile(out, files[rel], 0o644); err != nil {
			return 0, fmt.Errorf("write %s: %w", out, err)
		}
	}
	return len(files), nil
}

func pickAgent() agentTarget {
	detected := detectAgents()

	switch len(detected) {
	case 0:
		fmt.Println()
		fmt.Println("  No AI coding agent detected. Pick one:")
		fmt.Println()
		for i, a := range agents {
			fmt.Printf("    %d. %s\n", i+1, a.name)
		}
		fmt.Println()
		fmt.Print("  Enter number [1]: ")
		return readChoice(agents)

	case 1:
		fmt.Println()
		fmt.Printf("  Detected %s\n", detected[0].name)
		return detected[0]

	default:
		fmt.Println()
		fmt.Println("  Detected AI coding agents:")
		fmt.Println()
		for i, a := range detected {
			fmt.Printf("    %d. %s\n", i+1, a.name)
		}
		fmt.Println()
		fmt.Print("  Enter number [1]: ")
		return readChoice(detected)
	}
}

func detectAgents() []agentTarget {
	var found []agentTarget
	for _, a := range agents {
		if a.detect() {
			found = append(found, a)
		}
	}
	return found
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func hasDir(name string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(home, name)); err == nil {
		return true
	}
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

func readChoice(list []agentTarget) agentTarget {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return list[0]
	}

	var n int
	if _, err := fmt.Sscanf(input, "%d", &n); err != nil || n < 1 || n > len(list) {
		return list[0]
	}
	return list[n-1]
}

func printNextSteps() {
	fmt.Println("  Next steps:")
	fmt.Printf("  - Skills are auto-discovered at session start\n")
	fmt.Printf("  - Ask your agent to manage learning with pharos CLI\n")
	fmt.Printf("  - Run 'pharos skills uninstall' to remove installed skills\n")
}

func offerSkillInstall() {
	detected := detectAgents()
	if len(detected) == 0 {
		fmt.Println()
		fmt.Print("  No AI coding agent detected. Install the pharos skills anyway? [y/N] ")
		if promptYes() {
			installForAgent(pickAgent(), false)
		}
		return
	}

	// Fast-path: skip if all detected agents already have current skills.
	if allCurrent := skillsCurrent(detected); allCurrent {
		return
	}

	fmt.Println()
	if len(detected) == 1 {
		fmt.Printf("  Detected %s — install the pharos teaching skill? [Y/n] ", detected[0].name)
	} else {
		fmt.Print("  Install the pharos teaching skill for your AI coding agent? [Y/n] ")
	}
	if !promptDefaultYes() {
		return
	}
	installForAgent(pickAgent(), false)
}

func installForAgent(a agentTarget, project bool) {
	baseDir := a.installDir(project)
	n, err := installAllSkills(baseDir)
	if err != nil {
		fmt.Printf("  Warning: skill install failed: %v\n", err)
		return
	}
	fmt.Println()
	fmt.Printf("  ✓ Installed %d skill(s) to %s/ (%d files)\n", len(skills.All), baseDir, n)
	printNextSteps()
}

// skillsCurrent returns true when every detected agent has current skills at
// both global and project scopes.
func skillsCurrent(detected []agentTarget) bool {
	for _, a := range detected {
		for _, project := range []bool{false, true} {
			baseDir := a.installDir(project)
			if !isSkillInstalled(baseDir) {
				return false
			}
			for _, skill := range skills.All {
				embedded, err := skillFilesMap(skill)
				if err != nil {
					return false
				}
				if anySkillChanged(baseDir, skill, embedded) {
					return false
				}
			}
		}
	}
	return true
}

// skillFilesMap returns the embedded files for a single skill, keyed by
// path relative to the skill root.
func skillFilesMap(skill string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	prefix := skill + "/"
	err := fs.WalkDir(skills.Files, skill, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, prefix)
		data, err := skills.Files.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	return files, err
}

// anySkillChanged reports whether the installed copy of `skill` under baseDir
// differs from the embedded version. Uses the manifest hash for a fast
// comparison, then verifies all listed files still exist on disk.
func anySkillChanged(baseDir, skill string, embedded map[string][]byte) bool {
	dir := filepath.Join(baseDir, skill)
	m, err := skills.ReadManifest(dir)
	if err != nil {
		return true
	}
	if m.Hash != skills.ManifestHash(embedded) {
		return true
	}
	for _, f := range m.Files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return true
		}
	}
	return false
}

func offerSkillUpgrade() {
	// Check each embedded skill against each agent that has the primary skill,
	// checking both global and project-level installs.
	var outdated []agentTarget
	seen := map[string]bool{}
	for _, a := range agents {
		for _, project := range []bool{false, true} {
			baseDir := a.installDir(project)
			if !isSkillInstalled(baseDir) {
				continue
			}
			for _, skill := range skills.All {
				embedded, err := skillFilesMap(skill)
				if err != nil {
					continue
				}
				if anySkillChanged(baseDir, skill, embedded) {
					if !seen[a.name] {
						outdated = append(outdated, a)
						seen[a.name] = true
					}
				}
			}
		}
	}

	if len(outdated) == 0 {
		return
	}

	fmt.Println()
	if len(outdated) == 1 {
		fmt.Printf("  The pharos skills for %s have changed. Update them? [Y/n] ", outdated[0].name)
	} else {
		names := make([]string, len(outdated))
		for i, a := range outdated {
			names[i] = a.name
		}
		fmt.Printf("  Pharos skills have changed for %s. Update them? [Y/n] ", strings.Join(names, ", "))
	}

	if !promptDefaultYes() {
		return
	}

	for _, a := range outdated {
		for _, project := range []bool{false, true} {
			baseDir := a.installDir(project)
			if !isSkillInstalled(baseDir) {
				continue
			}
			n, err := installAllSkills(baseDir)
			if err != nil {
				fmt.Printf("  Warning: failed to update skills for %s: %v\n", a.name, err)
			} else {
				fmt.Printf("  ✓ Updated %s skills (%d files)\n", a.name, n)
			}
		}
	}
}

// isSkillInstalled reports whether the primary skill is present under baseDir.
func isSkillInstalled(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, skills.SkillName, "SKILL.md"))
	return err == nil
}

// appendIfMissing appends a to list only if it isn't already present.
func appendIfMissing(list []agentTarget, a agentTarget) []agentTarget {
	for _, e := range list {
		if e.name == a.name {
			return list
		}
	}
	return append(list, a)
}

func promptYes() bool {
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func promptDefaultYes() bool {
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}
