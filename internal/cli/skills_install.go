package cli

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/udit-001/pharos/internal/skills"
	"github.com/spf13/cobra"
)

type agentTarget struct {
	name    string
	subdir  string // relative path under the install root, e.g. ".opencode/skills"
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
	{name: "pi.dev", subdir: ".pi/skills", detect: func() bool { return hasBinary("pi") || hasDir(".pi") }},
}

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	agent, _ := cmd.Flags().GetString("agent")
	project, _ := cmd.Flags().GetBool("project")

	var selected agentTarget
	if agent != "" {
		for _, a := range agents {
			if a.name == agent {
				selected = a
				break
			}
		}
		if selected.name == "" {
			return fmt.Errorf("unknown agent %q\n  Supported: opencode, claude-code, codex, pi.dev", agent)
		}
	} else {
		selected = pickAgent()
		project = promptScope()
	}

	baseDir := selected.installDir(project)

	// Skip overwrite confirmation when --agent is given (non-interactive)
	if agent == "" {
		primaryDir := filepath.Join(baseDir, skills.SkillName)
		if _, err := os.Stat(primaryDir); err == nil {
			fmt.Printf("  %s/ already exists.\n", primaryDir)
			fmt.Print("  Overwrite? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("  Skipped.")
				return nil
			}
		}
	}

	n, err := installAllSkills(baseDir)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  ✓ Installed %d skill(s) to %s/ (%d files)\n", len(skills.All), baseDir, n)
	fmt.Println()
	printNextSteps()
	fmt.Println()
	return nil
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

// installAllSkills installs every embedded skill (learn, teach, ...) into
// baseDir/<skillName>. Returns the total file count across all skills.
func installAllSkills(baseDir string) (int, error) {
	total := 0
	for _, skill := range skills.All {
		n, err := installSkillDir(skills.Files, skill, filepath.Join(baseDir, skill))
		if err != nil {
			return total, fmt.Errorf("install %s skill: %w", skill, err)
		}
		total += n
	}
	return total, nil
}

// installSkillDir walks the embedded srcDir and writes every file under destDir.
func installSkillDir(fsys embed.FS, srcDir, destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create directories: %w", err)
	}

	count := 0
	err := fs.WalkDir(fsys, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		data, err := fsys.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		out := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", out, err)
		}
		count++
		return nil
	})
	if err != nil {
		return count, fmt.Errorf("install skill: %w", err)
	}
	return count, nil
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
}

func offerSkillInstall() {
	detected := detectAgents()

	install := func(a agentTarget, project bool) {
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

	switch len(detected) {
	case 1:
		fmt.Println()
		fmt.Printf("  Detected %s — installing pharos skills...\n", detected[0].name)
		install(detected[0], false) // install globally during init
	case 0:
		fmt.Println()
		fmt.Print("  No AI coding agent detected. Install the pharos skills anyway? [y/N] ")
		if promptYes() {
			install(pickAgent(), false)
		}
	default:
		fmt.Println()
		fmt.Print("  Install the pharos skills for an AI coding agent? [Y/n] ")
		if promptDefaultYes() {
			install(pickAgent(), false)
		}
	}
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
// differs from the embedded version (missing, extra, or differing files).
func anySkillChanged(baseDir, skill string, embedded map[string][]byte) bool {
	dir := filepath.Join(baseDir, skill)
	for rel, want := range embedded {
		got, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return true
		}
		if string(got) != string(want) {
			return true
		}
	}
	installedCount := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			installedCount++
		}
		return nil
	})
	return installedCount != len(embedded)
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
