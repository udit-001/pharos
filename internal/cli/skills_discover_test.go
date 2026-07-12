package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func mkSkillDirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(d, "teach"), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
}

// Global + project locations are discovered.
func TestDiscoverAt_GlobalAndProject(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(base, "proj")
	homeAgents := filepath.Join(home, ".agents", "skills")
	cwdAgents := filepath.Join(cwd, ".agents", "skills")
	mkSkillDirs(t, homeAgents, cwdAgents)

	locs := discoverAt(home, cwd)

	if len(locs) != 2 {
		t.Fatalf("expected 2 locs, got %d: %+v", len(locs), locs)
	}
	if locs[0].dir != homeAgents || locs[0].scope != "global" {
		t.Errorf("loc[0] = %+v, want global %s", locs[0], homeAgents)
	}
	if locs[1].dir != cwdAgents || locs[1].scope != "project" {
		t.Errorf("loc[1] = %+v, want project %s", locs[1], cwdAgents)
	}
}

// Both families (.agents/skills + .claude/skills) are discovered.
func TestDiscoverAt_BothFamilies(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(base, "proj")
	homeAgents := filepath.Join(home, ".agents", "skills")
	homeClaude := filepath.Join(home, ".claude", "skills")
	mkSkillDirs(t, homeAgents, homeClaude)

	locs := discoverAt(home, cwd)

	if len(locs) != 2 {
		t.Fatalf("expected 2 locs, got %d: %+v", len(locs), locs)
	}
	families := map[string]bool{}
	for _, l := range locs {
		families[l.family] = true
	}
	if !families["agents"] || !families["claude"] {
		t.Errorf("expected both families, got: %+v", locs)
	}
}

// No ancestor walk — dirs above CWD are not discovered.
func TestDiscoverAt_NoAncestorWalk(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "Dev", "random")
	ancestor := filepath.Join(home, "Dev")
	ancestorAgents := filepath.Join(ancestor, ".agents", "skills")
	homeAgents := filepath.Join(home, ".agents", "skills")
	mkSkillDirs(t, ancestorAgents, homeAgents)

	locs := discoverAt(home, cwd)

	for _, l := range locs {
		if l.dir == ancestorAgents {
			t.Fatalf("ancestor dir should not be discovered: %+v", l)
		}
	}
	if len(locs) != 1 || locs[0].dir != homeAgents {
		t.Fatalf("expected only global home, got: %+v", locs)
	}
}

// No skills dirs yields no locations.
func TestDiscoverAt_Empty(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(base, "proj")

	locs := discoverAt(home, cwd)
	if len(locs) != 0 {
		t.Fatalf("expected 0 locs, got %d: %+v", len(locs), locs)
	}
}

// Project dir that equals home dir is not duplicated.
func TestDiscoverAt_DedupsHomeAndProject(t *testing.T) {
	base := t.TempDir()
	home := base
	agentsSkills := filepath.Join(home, ".agents", "skills")
	mkSkillDirs(t, agentsSkills)

	locs := discoverAt(home, home)

	count := 0
	for _, l := range locs {
		if l.dir == agentsSkills {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("home=project dir appears %d times, want 1: %+v", count, locs)
	}
}
