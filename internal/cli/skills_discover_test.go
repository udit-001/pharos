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

// Non-git: walk continues to the filesystem root, so an ancestor
// .opencode/skills (the stale-shadow scenario) is discovered, and
// home is collected in the walk (tagged global) rather than appended
// twice.
func TestDiscoverFrom_NonGitWalksToRoot(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "Dev", "random")
	ancestor := filepath.Join(home, "Dev")
	ancestorSkills := filepath.Join(ancestor, ".opencode", "skills")
	homeSkills := filepath.Join(home, ".agents", "skills")
	mkSkillDirs(t, ancestorSkills, homeSkills)

	subdirs := map[string][]string{
		".opencode/skills": {"opencode"},
		".agents/skills":   {"opencode", "codex", "pi.dev"},
	}
	locs := discoverFrom(subdirs, cwd, "", home)

	if len(locs) != 2 {
		t.Fatalf("expected 2 locs, got %d: %+v", len(locs), locs)
	}
	if locs[0].subdir != ".opencode/skills" || locs[0].scope != "ancestor" || locs[0].dir != ancestorSkills {
		t.Errorf("loc[0] = %+v, want ancestor .opencode/skills at %s", locs[0], ancestorSkills)
	}
	if locs[1].subdir != ".agents/skills" || locs[1].scope != "global" || locs[1].dir != homeSkills {
		t.Errorf("loc[1] = %+v, want global .agents/skills at %s", locs[1], homeSkills)
	}
}

// Git repo: walk stops at the worktree root, so an ancestor above the
// worktree is NOT discovered. The global home dir is still reachable
// via the final append.
func TestDiscoverFrom_GitStopsAtWorktree(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	repo := filepath.Join(base, "repo")
	cwd := filepath.Join(repo, "sub", "deep")
	repoSkills := filepath.Join(repo, ".agents", "skills")
	homeSkills := filepath.Join(home, ".agents", "skills")
	mkSkillDirs(t, repoSkills, homeSkills)

	subdirs := map[string][]string{
		".agents/skills": {"opencode", "codex", "pi.dev"},
	}
	locs := discoverFrom(subdirs, cwd, repo, home)

	if len(locs) != 2 {
		t.Fatalf("expected 2 locs, got %d: %+v", len(locs), locs)
	}
	if locs[0].dir != repoSkills {
		t.Errorf("loc[0].dir = %s, want %s", locs[0].dir, repoSkills)
	}
	if locs[1].scope != "global" || locs[1].dir != homeSkills {
		t.Errorf("loc[1] = %+v, want global %s", locs[1], homeSkills)
	}
}

// When the CWD itself has a skills dir, it is tagged "project".
func TestDiscoverFrom_CWDIsProject(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "proj")
	cwdSkills := filepath.Join(cwd, ".agents", "skills")
	mkSkillDirs(t, cwdSkills)

	subdirs := map[string][]string{
		".agents/skills": {"opencode", "codex", "pi.dev"},
	}
	locs := discoverFrom(subdirs, cwd, "", base)

	found := false
	for _, l := range locs {
		if l.dir == cwdSkills && l.scope != "project" {
			t.Errorf("cwd scope = %q, want project", l.scope)
		}
		if l.dir == cwdSkills {
			found = true
		}
	}
	if !found {
		t.Fatalf("cwd skills dir not discovered: %+v", locs)
	}
}

// Home reached during the walk must not be appended a second time.
func TestDiscoverFrom_DedupsHomeReachedInWalk(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "a", "b")
	homeSkills := filepath.Join(base, ".agents", "skills")
	mkSkillDirs(t, homeSkills)

	subdirs := map[string][]string{
		".agents/skills": {"opencode", "codex", "pi.dev"},
	}
	locs := discoverFrom(subdirs, cwd, "", base)

	count := 0
	for _, l := range locs {
		if l.dir == homeSkills {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("home dir appears %d times, want 1: %+v", count, locs)
	}
}

// Multiple subdirs at the same ancestor are all discovered.
func TestDiscoverFrom_MultipleSubdirs(t *testing.T) {
	base := t.TempDir()
	ancestor := filepath.Join(base, "dev")
	cwd := filepath.Join(ancestor, "proj")
	opencodeSkills := filepath.Join(ancestor, ".opencode", "skills")
	agentsSkills := filepath.Join(ancestor, ".agents", "skills")
	claudeSkills := filepath.Join(ancestor, ".claude", "skills")
	mkSkillDirs(t, opencodeSkills, agentsSkills, claudeSkills)

	subdirs := map[string][]string{
		".opencode/skills": {"opencode"},
		".agents/skills":   {"opencode", "codex", "pi.dev"},
		".claude/skills":   {"opencode", "claude-code"},
	}
	locs := discoverFrom(subdirs, cwd, "", base)

	if len(locs) != 3 {
		t.Fatalf("expected 3 locs, got %d: %+v", len(locs), locs)
	}
}

// No skills dirs anywhere yields no locations.
func TestDiscoverFrom_Empty(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "a", "b")
	subdirs := map[string][]string{
		".agents/skills": {"opencode"},
	}
	locs := discoverFrom(subdirs, cwd, "", base)
	if len(locs) != 0 {
		t.Fatalf("expected 0 locs, got %d: %+v", len(locs), locs)
	}
}

// Readers from multiple providers that share a subdir are merged.
func TestDiscoverFrom_MergesReaders(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "proj")
	homeAgents := filepath.Join(home, ".agents", "skills")
	mkSkillDirs(t, homeAgents)

	subdirs := map[string][]string{
		".agents/skills": {"opencode", "codex", "pi.dev"},
	}
	locs := discoverFrom(subdirs, cwd, "", home)

	if len(locs) != 1 {
		t.Fatalf("expected 1 loc, got %d: %+v", len(locs), locs)
	}
	if len(locs[0].readers) != 3 {
		t.Errorf("expected 3 readers, got %d: %+v", len(locs[0].readers), locs[0].readers)
	}
}
