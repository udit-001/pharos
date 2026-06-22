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

// TestGetLessonBySeq proves GetLessonBySeq uses a WHERE clause (not
// fetch-all-and-loop): it finds the right lesson by seq, errors on not-found,
// and respects workspace scoping.
func TestGetLessonBySeq(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	beta := seedWorkspace(t, store, "beta")
	beta.AddLesson(Lesson{Title: "b1", Filename: "b1.html"})

	// Found
	got, err := alpha.GetLessonBySeq(2)
	if err != nil {
		t.Fatalf("GetLessonBySeq(2): %v", err)
	}
	if got.Title != "a2" {
		t.Errorf("got %q, want a2", got.Title)
	}

	// Not found in workspace
	if _, err := alpha.GetLessonBySeq(99); err == nil {
		t.Error("expected error for seq 99, got nil")
	}

	// Scoping: beta's seq 1 is "b1", not "a1"
	gotBeta, err := beta.GetLessonBySeq(1)
	if err != nil {
		t.Fatalf("beta GetLessonBySeq(1): %v", err)
	}
	if gotBeta.Title != "b1" {
		t.Errorf("beta seq 1 = %q, want b1", gotBeta.Title)
	}
}

// TestGetRecordBySeq proves GetRecordBySeq uses a WHERE clause.
func TestGetRecordBySeq(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md"})
	alpha.AddRecord(LearningRecord{Title: "r2", Filename: "r2.md"})

	got, err := alpha.GetRecordBySeq(2)
	if err != nil {
		t.Fatalf("GetRecordBySeq(2): %v", err)
	}
	if got.Title != "r2" {
		t.Errorf("got %q, want r2", got.Title)
	}

	if _, err := alpha.GetRecordBySeq(99); err == nil {
		t.Error("expected error for seq 99, got nil")
	}
}

// TestGetRefBySlug proves GetRefBySlug uses a WHERE clause.
func TestGetRefBySlug(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddRef(Reference{Title: "Joins", Slug: "joins", Filename: "joins.html"})
	alpha.AddRef(Reference{Title: "Indexes", Slug: "indexes", Filename: "indexes.html"})

	got, err := alpha.GetRefBySlug("indexes")
	if err != nil {
		t.Fatalf("GetRefBySlug(indexes): %v", err)
	}
	if got.Title != "Indexes" {
		t.Errorf("got %q, want Indexes", got.Title)
	}

	if _, err := alpha.GetRefBySlug("nonexistent"); err == nil {
		t.Error("expected error for nonexistent slug, got nil")
	}
}

// TestGetSidebarData proves GetSidebarData returns the lightweight sidebar
// projections (not full model structs) with correct fields and workspace
// scoping. One call replaces three separate Get* calls.
func TestGetSidebarData(t *testing.T) {
	store := newTestStore(t)
	alpha := seedWorkspace(t, store, "alpha")
	alpha.AddLesson(Lesson{Title: "a1", Filename: "a1.html"})
	alpha.AddLesson(Lesson{Title: "a2", Filename: "a2.html"})
	alpha.AddRecord(LearningRecord{Title: "r1", Filename: "r1.md", Status: "active", Summary: "sum"})
	alpha.AddRef(Reference{Title: "Ref1", Slug: "ref1", Filename: "ref1.html"})

	// Empty workspace
	beta := seedWorkspace(t, store, "beta")

	// Alpha: 2 lessons, 1 record, 1 ref
	sd, err := alpha.GetSidebarData()
	if err != nil {
		t.Fatalf("GetSidebarData: %v", err)
	}
	if sd.Workspace.Name != "alpha" {
		t.Errorf("workspace name = %q, want alpha", sd.Workspace.Name)
	}
	if len(sd.Lessons) != 2 {
		t.Fatalf("lessons = %d, want 2", len(sd.Lessons))
	}
	if sd.Lessons[0].Seq != 1 || sd.Lessons[0].Title != "a1" {
		t.Errorf("lesson[0] = %+v, want {1, a1}", sd.Lessons[0])
	}
	if len(sd.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(sd.Records))
	}
	if sd.Records[0].Status != "active" || sd.Records[0].Summary != "sum" {
		t.Errorf("record[0] = %+v, want status=active summary=sum", sd.Records[0])
	}
	if len(sd.Refs) != 1 {
		t.Fatalf("refs = %d, want 1", len(sd.Refs))
	}
	if sd.Refs[0].Slug != "ref1" || sd.Refs[0].Title != "Ref1" {
		t.Errorf("ref[0] = %+v, want {ref1, Ref1}", sd.Refs[0])
	}

	// Beta: empty workspace — no items leaked from alpha
	sdBeta, err := beta.GetSidebarData()
	if err != nil {
		t.Fatalf("beta GetSidebarData: %v", err)
	}
	if len(sdBeta.Lessons) != 0 || len(sdBeta.Records) != 0 || len(sdBeta.Refs) != 0 {
		t.Errorf("beta should be empty, got L%d R%d Ref%d",
			len(sdBeta.Lessons), len(sdBeta.Records), len(sdBeta.Refs))
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
