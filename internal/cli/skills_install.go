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

// --- Types ---

// provider is a supported AI coding agent and the directories it reads
// skills from. Discovery uses readSubdirs to find every copy on disk;
// detection narrows which providers are present on this machine.
type provider struct {
	name        string
	aliases     []string
	readSubdirs []string
	detect      func() bool
}

// installFamily is a write target: a directory name + which providers
// read it. readers is for display ("read by"); installFor is which
// detected providers trigger writing to this family.
type installFamily struct {
	name       string
	subdir     string
	readers    []string
	installFor []string
}

// skillLocation is a discovered skill directory with its classification.
type skillLocation struct {
	dir       string
	subdir    string
	scope     string
	readers   []string
	family    string
	status    string
	unmanaged bool
}

// --- Vars ---

var providers = []provider{
	{
		name:        "opencode",
		readSubdirs: []string{".opencode/skills", ".claude/skills", ".agents/skills"},
		detect:      func() bool { return hasBinary("opencode") || hasDir(".opencode") },
	},
	{
		name:        "codex",
		readSubdirs: []string{".codex/skills", ".agents/skills"},
		detect:      func() bool { return hasBinary("codex") || hasDir(".codex") },
	},
	{
		name:        "pi.dev",
		aliases:     []string{"pi"},
		readSubdirs: []string{".pi/skills", ".agents/skills"},
		detect:      func() bool { return hasBinary("pi") || hasDir(".pi") },
	},
	{
		name:        "claude-code",
		aliases:     []string{"claude"},
		readSubdirs: []string{".claude/skills"},
		detect:      func() bool { return hasBinary("claude") || hasDir(".claude") },
	},
}

var families = []installFamily{
	{
		name:       "agents",
		subdir:     ".agents/skills",
		readers:    []string{"opencode", "codex", "pi.dev"},
		installFor: []string{"opencode", "codex", "pi.dev"},
	},
	{
		name:       "claude",
		subdir:     ".claude/skills",
		readers:    []string{"opencode", "claude-code"},
		installFor: []string{"claude-code"},
	},
}

// --- Discovery ---

func discover() []skillLocation {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	cwd, _ = filepath.Abs(cwd)
	home, _ := os.UserHomeDir()
	stop := ""
	if root, err := gitWorktreeRoot(cwd); err == nil {
		stop = root
	}

	subdirReaders := map[string][]string{}
	for _, p := range providers {
		for _, sd := range p.readSubdirs {
			subdirReaders[sd] = appendUnique(subdirReaders[sd], p.name)
		}
	}

	locs := discoverFrom(subdirReaders, cwd, stop, home)

	embedded, _ := skillFilesMap(skills.SkillName)
	for i := range locs {
		classifyLocation(&locs[i], embedded)
	}
	return locs
}

func appendUnique(list []string, val string) []string {
	for _, v := range list {
		if v == val {
			return list
		}
	}
	return append(list, val)
}

func discoverFrom(subdirs map[string][]string, start, stop, home string) []skillLocation {
	var locs []skillLocation
	seen := map[string]bool{}

	sortedSubdirs := make([]string, 0, len(subdirs))
	for sd := range subdirs {
		sortedSubdirs = append(sortedSubdirs, sd)
	}
	sort.Strings(sortedSubdirs)

	dir := start
	for {
		for _, sd := range sortedSubdirs {
			path := filepath.Join(dir, sd)
			if !dirExists(path) || seen[path] {
				continue
			}
			scope := "ancestor"
			switch {
			case dir == start:
				scope = "project"
			case home != "" && dir == home:
				scope = "global"
			}
			locs = append(locs, skillLocation{
				dir:     path,
				subdir:  sd,
				scope:   scope,
				readers: subdirs[sd],
			})
			seen[path] = true
		}
		if stop != "" && dir == stop {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if home != "" {
		for _, sd := range sortedSubdirs {
			path := filepath.Join(home, sd)
			if !seen[path] && dirExists(path) {
				locs = append(locs, skillLocation{
					dir:     path,
					subdir:  sd,
					scope:   "global",
					readers: subdirs[sd],
				})
				seen[path] = true
			}
		}
	}
	return locs
}

func classifyLocation(loc *skillLocation, embedded map[string][]byte) {
	loc.family = ""
	for _, f := range families {
		if loc.subdir == f.subdir {
			loc.family = f.name
			break
		}
	}
	if loc.family == "" {
		loc.status = "orphan"
	} else {
		if anySkillChanged(loc.dir, skills.SkillName, embedded) {
			loc.status = "outdated"
		} else {
			loc.status = "current"
		}
	}
	manifestPath := skills.ManifestPath(filepath.Join(loc.dir, skills.SkillName))
	if _, err := os.Stat(manifestPath); err != nil {
		loc.unmanaged = true
	}
}

func gitWorktreeRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// --- Detection ---

func detectProviders() []provider {
	var found []provider
	for _, p := range providers {
		if p.detect() {
			found = append(found, p)
		}
	}
	return found
}

func familiesForProviders(detected []provider) []installFamily {
	var result []installFamily
	for _, f := range families {
		for _, name := range f.installFor {
			for _, p := range detected {
				if p.name == name {
					result = append(result, f)
					break
				}
			}
		}
	}
	return result
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

// --- Install ---

func runSkillsInstall(cmd *cobra.Command, args []string) error {
	agentsOnly, _ := cmd.Flags().GetBool("agents-only")
	claudeOnly, _ := cmd.Flags().GetBool("claude-only")
	all, _ := cmd.Flags().GetBool("all")
	project, _ := cmd.Flags().GetBool("project")

	if all && (agentsOnly || claudeOnly) {
		return fmt.Errorf("--all is mutually exclusive with --agents-only and --claude-only")
	}
	if agentsOnly && claudeOnly {
		return fmt.Errorf("use --all to install both families, not --agents-only and --claude-only together")
	}

	nonInteractive := all || agentsOnly || claudeOnly

	detected := detectProviders()
	if len(detected) == 0 {
		if !nonInteractive {
			fmt.Println()
			fmt.Print("  No AI coding agent detected. Install the pharos skills anyway? [y/N] ")
			if !promptYes() {
				return nil
			}
		}
		return installFamilies(families, project)
	}

	names := make([]string, len(detected))
	for i, p := range detected {
		names[i] = p.name
	}
	fmt.Println()
	fmt.Printf("  Detected: %s\n", strings.Join(names, ", "))

	var selectedFamilies []installFamily
	switch {
	case agentsOnly:
		selectedFamilies = []installFamily{families[0]}
	case claudeOnly:
		selectedFamilies = []installFamily{families[1]}
	case all:
		selectedFamilies = familiesForProviders(detected)
		if len(selectedFamilies) == 0 {
			fmt.Println("  No installable families for detected providers.")
			return nil
		}
	default:
		avail := familiesForProviders(detected)
		if len(avail) == 0 {
			fmt.Println("  No installable families for detected providers.")
			return nil
		}
		if len(avail) <= 1 {
			selectedFamilies = avail
		} else {
			selectedFamilies = promptFamilySelect(avail)
			if selectedFamilies == nil {
				fmt.Println("  Cancelled.")
				return nil
			}
		}
	}

	if !nonInteractive && !project {
		var cancelled bool
		project, cancelled = promptInstallScope(selectedFamilies)
		if cancelled {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	return installFamilies(selectedFamilies, project)
}

func promptFamilySelect(avail []installFamily) []installFamily {
	fmt.Println()
	fmt.Println("  Install to:")
	fmt.Printf("    1. Both           — %s, %s\n", familyGlobalDir(families[0]), familyGlobalDir(families[1]))
	fmt.Printf("    2. Standard only  — %s  (%s)\n", familyGlobalDir(families[0]), strings.Join(families[0].readers, ", "))
	fmt.Printf("    3. Claude only    — %s  (%s)\n", familyGlobalDir(families[1]), strings.Join(families[1].readers, ", "))
	fmt.Println("    0. Cancel")
	fmt.Println()
	fmt.Print("  Enter number [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "0":
		return nil
	case "2":
		return []installFamily{families[0]}
	case "3":
		return []installFamily{families[1]}
	default:
		return avail
	}
}

func promptInstallScope(selectedFamilies []installFamily) (bool, bool) {
	globalDirs := make([]string, len(selectedFamilies))
	projectDirs := make([]string, len(selectedFamilies))
	for i, f := range selectedFamilies {
		globalDirs[i] = familyGlobalDir(f)
		projectDirs[i] = "./" + f.subdir
	}
	fmt.Println()
	fmt.Println("  Scope:")
	fmt.Printf("    1. Globally     — %s\n", strings.Join(globalDirs, ", "))
	fmt.Printf("    2. This project — %s\n", strings.Join(projectDirs, ", "))
	fmt.Println("    0. Cancel")
	fmt.Println()
	fmt.Print("  Enter number [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "0":
		return false, true
	case "2":
		return true, false
	default:
		return false, false
	}
}

func familyGlobalDir(f installFamily) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return f.subdir
	}
	return filepath.Join(home, f.subdir)
}

func familyDir(f installFamily, project bool) string {
	if project {
		return f.subdir
	}
	return familyGlobalDir(f)
}

func installFamilies(selectedFamilies []installFamily, project bool) error {
	var errors []string
	for _, f := range families {
		if !familyInList(f, selectedFamilies) {
			continue
		}
		baseDir := familyDir(f, project)
		action := "Installed"
		if isSkillInstalled(baseDir) {
			if anySkillChangedForAll(baseDir) {
				action = "Updated"
			} else {
				fmt.Printf("  ✓ Already current at %s (%s)\n", baseDir, strings.Join(f.readers, ", "))
				continue
			}
		}
		n, err := installAllSkills(baseDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", f.name, err))
			continue
		}
		fmt.Printf("  ✓ %s teach to %s/ (%d files) — %s\n", action, baseDir, n, strings.Join(f.readers, ", "))
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

func familyInList(f installFamily, list []installFamily) bool {
	for _, x := range list {
		if x.name == f.name {
			return true
		}
	}
	return false
}

// installAllSkills installs every embedded skill into baseDir/<skillName>,
// then writes a manifest for change detection. Returns total file count.
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

// --- Skill content helpers ---

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

func anySkillChangedForAll(baseDir string) bool {
	for _, skill := range skills.All {
		embedded, err := skillFilesMap(skill)
		if err != nil {
			return true
		}
		if anySkillChanged(baseDir, skill, embedded) {
			return true
		}
	}
	return false
}

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

func isSkillInstalled(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, skills.SkillName, "SKILL.md"))
	return err == nil
}

// --- Startup hooks ---

func offerSkillInstall() {
	detected := detectProviders()
	if len(detected) == 0 {
		return
	}
	avail := familiesForProviders(detected)
	if len(avail) == 0 {
		return
	}
	allCurrent := true
	for _, f := range avail {
		baseDir := familyGlobalDir(f)
		if !isSkillInstalled(baseDir) || anySkillChangedForAll(baseDir) {
			allCurrent = false
			break
		}
	}
	if allCurrent {
		return
	}
	fmt.Println()
	names := make([]string, len(detected))
	for i, p := range detected {
		names[i] = p.name
	}
	if len(detected) == 1 {
		fmt.Printf("  Detected %s — install the pharos teaching skill? [Y/n] ", detected[0].name)
	} else {
		fmt.Printf("  Detected %s — install the pharos teaching skill? [Y/n] ", strings.Join(names, ", "))
	}
	if !promptDefaultYes() {
		return
	}
	installFamilies(avail, false)
}

func offerSkillUpgrade() {
	locs := discover()

	type staleLoc struct {
		loc skillLocation
	}
	var outdated []staleLoc
	var orphans []staleLoc

	for _, loc := range locs {
		if !isSkillInstalled(loc.dir) {
			continue
		}
		switch loc.status {
		case "outdated":
			outdated = append(outdated, staleLoc{loc})
		case "orphan":
			orphans = append(orphans, staleLoc{loc})
		}
	}

	if len(outdated) == 0 && len(orphans) == 0 {
		return
	}

	fmt.Println()
	if len(orphans) > 0 {
		fmt.Println("  Orphaned skill installs found:")
		for _, o := range orphans {
			fmt.Printf("    ⚠ %s\n", formatLocationLine(o.loc))
		}
		fmt.Print("  Remove orphaned installs? [Y/n] ")
		if promptDefaultYes() {
			for _, o := range orphans {
				if err := uninstallSkills(o.loc.dir); err != nil {
					fmt.Printf("  Warning: could not remove %s: %v\n", o.loc.dir, err)
				} else {
					fmt.Printf("  ✓ Removed %s\n", o.loc.dir)
				}
			}
		}
		fmt.Println()
	}

	if len(outdated) > 0 {
		fmt.Println("  Outdated skill installs found:")
		for _, o := range outdated {
			fmt.Printf("    ⚠ %s\n", formatLocationLine(o.loc))
		}
		fmt.Print("  Update them? [Y/n] ")
		if promptDefaultYes() {
			for _, o := range outdated {
				n, err := installAllSkills(o.loc.dir)
				if err != nil {
					fmt.Printf("  Warning: failed to update %s: %v\n", o.loc.dir, err)
				} else {
					fmt.Printf("  ✓ Updated %s (%d files)\n", o.loc.dir, n)
				}
			}
		}
	}
}

// formatLocationLine renders a skillLocation for human-readable output.
func formatLocationLine(loc skillLocation) string {
	parts := []string{loc.dir, "—", loc.status, "—", loc.scope}
	if len(loc.readers) > 0 {
		parts = append(parts, "—", strings.Join(loc.readers, ", "))
	} else {
		parts = append(parts, "— (no agent reads this path)")
	}
	if sw := shadowWarning(loc); sw != "" {
		parts = append(parts, sw)
	}
	if loc.unmanaged {
		parts = append(parts, "[unmanaged]")
	}
	return strings.Join(parts, " ")
}

// shadowWarning returns a parenthetical warning if an orphan location
// shadows a target install for any of its readers.
func shadowWarning(loc skillLocation) string {
	if loc.status != "orphan" || len(loc.readers) == 0 {
		return ""
	}
	for _, reader := range loc.readers {
		for _, p := range providers {
			if p.name != reader {
				continue
			}
			for _, sd := range p.readSubdirs {
				for _, f := range families {
					if sd == f.subdir {
						return fmt.Sprintf("(shadows %s for %s)", familyGlobalDir(f), reader)
					}
				}
			}
		}
	}
	return ""
}

// --- Prompts ---

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

func printNextSteps() {
	fmt.Println("  Next steps:")
	fmt.Printf("  - Skills are auto-discovered at session start\n")
	fmt.Printf("  - Ask your agent to manage learning with pharos CLI\n")
	fmt.Printf("  - Run 'pharos skills uninstall' to remove installed skills\n")
}
