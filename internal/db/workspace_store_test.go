package db

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// seedWorkspace creates a workspace and returns its WorkspaceStore for
// seeding items. This replaces direct store.AddLesson/AddLearningRecord/
// AddReference calls that used the now-removed flat Store item-methods.
func seedWorkspace(t *testing.T, store *Store, name string) *WorkspaceStore {
	t.Helper()
	_, err := store.AddWorkspace(Workspace{Name: name, Path: "/tmp/" + name})
	if err != nil {
		t.Fatalf("seed workspace %s: %v", name, err)
	}
	wsStore, err := store.Workspace(name)
	if err != nil {
		t.Fatalf("get workspace %s: %v", name, err)
	}
	return wsStore
}

// TestWorkspaceStoreScoping proves the WorkspaceStore seam: workspace
// resolution happens once at construction, scoped methods need no ID.
func TestWorkspaceStoreScoping(t *testing.T) {
	store := newTestStore(t)

	// Seed two workspaces with lessons
	alpha := seedWorkspace(t, store, "alpha")
	if _, err := alpha.AddLesson(Lesson{Title: "alpha-1", Filename: "a1.html"}); err != nil {
		t.Fatal(err)
	}
	beta := seedWorkspace(t, store, "beta")
	if _, err := beta.AddLesson(Lesson{Title: "beta-1", Filename: "b1.html"}); err != nil {
		t.Fatal(err)
	}

	// Scope to alpha — no ID passed from here on
	ws, err := store.Workspace("alpha")
	if err != nil {
		t.Fatalf("scope alpha: %v", err)
	}
	if got := ws.Workspace().Name; got != "alpha" {
		t.Errorf("scoped workspace name = %q, want alpha", got)
	}

	lessons, err := ws.GetLessons()
	if err != nil {
		t.Fatalf("scoped GetLessons: %v", err)
	}
	if len(lessons) != 1 || lessons[0].Title != "alpha-1" {
		t.Errorf("scoped lessons = %+v, want [alpha-1]", lessons)
	}

	// AddLesson via the scoped store — WorkspaceID set automatically
	created, err := ws.AddLesson(Lesson{Title: "alpha-2", Filename: "a2.html"})
	if err != nil {
		t.Fatalf("scoped AddLesson: %v", err)
	}
	if created.WorkspaceID != alpha.Workspace().ID {
		t.Errorf("AddLesson WorkspaceID = %d, want %d (should be auto-set)", created.WorkspaceID, alpha.Workspace().ID)
	}
	if created.SequenceNumber != 2 {
		t.Errorf("AddLesson SequenceNumber = %d, want 2", created.SequenceNumber)
	}

	// SetLastViewed + Touch — scoped, no ID
	if err := ws.SetLastViewed("lesson", 2); err != nil {
		t.Fatalf("scoped SetLastViewed: %v", err)
	}
	if err := ws.Touch(); err != nil {
		t.Fatalf("scoped Touch: %v", err)
	}

	// Cross-check: beta still has only 1 lesson (alpha-2 didn't leak)
	betaLessons, _ := beta.GetLessons()
	if len(betaLessons) != 1 {
		t.Errorf("beta lessons = %d, want 1 (scoping leak?)", len(betaLessons))
	}
}

// TestWorkspaceUnknownName proves the seam errors cleanly on bad input.
func TestWorkspaceUnknownName(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Workspace("does-not-exist"); err == nil {
		t.Error("expected error for unknown workspace, got nil")
	}
}

// TestGetWorkspacesCountsCorrect proves LEARN-17's batch enrichment: counts
// are correct across multiple workspaces with varying lesson/record/ref
// counts, via the grouped COUNT(*) queries (not the old per-workspace N+1).
func TestGetWorkspacesCountsCorrect(t *testing.T) {
	store := newTestStore(t)

	// alpha: 2 lessons, 1 record, 1 ref
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md"})
	alpha.AddRef(Reference{Title: "ref1", Filename: "ref1.html"})

	// beta: 0 lessons, 2 records, 0 refs
	beta := seedWorkspace(t, store, "beta")
	beta.AddRecord(LearningRecord{Title: "b1", Filename: "b1.md"})
	beta.AddRecord(LearningRecord{Title: "b2", Filename: "b2.md"})

	ws, err := store.GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces: %v", err)
	}
	byName := map[string]Workspace{}
	for _, w := range ws {
		byName[w.Name] = w
	}

	cases := []struct {
		name               string
		wantL, wantR, wantRef int
	}{
		{"alpha", 2, 1, 1},
		{"beta", 0, 2, 0},
	}
	for _, c := range cases {
		w := byName[c.name]
		if w.LessonCount != c.wantL || w.RecordCount != c.wantR || w.RefCount != c.wantRef {
			t.Errorf("%s: counts = L%d R%d Ref%d, want L%d R%d Ref%d",
				c.name, w.LessonCount, w.RecordCount, w.RefCount, c.wantL, c.wantR, c.wantRef)
		}
	}
}
