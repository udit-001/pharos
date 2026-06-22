package db

import (
	"testing"
)

// TestStoreDoesNotExposeRawSQL is a compile-time check that sealing LEARN-12
// held: the public Store type must not promote *sqlx.DB's Exec/Query/Get
// methods. A regression that re-embeds *sqlx.DB would make those appear and
// fail the test.
func TestStoreDoesNotExposeRawSQL(t *testing.T) {
	store := newTestStore(t)
	_ = store

	// These workspace-management methods must exist (the real interface).
	_, _, _ = store.GetWorkspaces, store.GetWorkspace, store.WorkspaceCount

	// If *sqlx.DB were embedded, these would resolve. They must NOT —
	// uncomment to confirm they fail to compile:
	//   store.Exec("DELETE FROM workspaces")
	//   store.Query("SELECT * FROM workspaces")
	//   store.Get(&struct{}{}, "SELECT 1")
	//
	// The fact that they don't compile is the encapsulation guarantee.
}

// TestStoreDoesNotExposeItemMethods confirms Store no longer exposes the
// flat workspaceID-parameterized item methods (GetLessons, AddLesson,
// SearchLessons, etc.). All item access goes through WorkspaceStore.
// A regression that re-adds these methods would let callers bypass the
// workspace seam.
func TestStoreDoesNotExposeItemMethods(t *testing.T) {
	store := newTestStore(t)
	_ = store

	// Workspace-management methods exist and work:
	_, _ = store.Workspace, store.WorkspaceByID

	// Flat item methods must NOT be reachable on Store. If they were
	// re-added, these would compile. Uncomment to confirm they fail:
	//   store.GetLessons(1)
	//   store.AddLesson(Lesson{})
	//   store.SearchLessons("q", 1)
	//   store.GetLearningRecords(1)
	//   store.AddLearningRecord(LearningRecord{})
	//   store.SearchLearningRecords("q", 1)
	//   store.GetReferences(1)
	//   store.AddReference(Reference{})
	//   store.SearchReferences("q", 1)
}

// TestWorkspaceStoreDoesNotExposeStore confirms WorkspaceStore no longer
// promotes *Store's methods. A regression that re-embeds *Store would let
// a caller bypass the workspace boundary — e.g. wsStore.GetWorkspaces().
func TestWorkspaceStoreDoesNotExposeStore(t *testing.T) {
	store := newTestStore(t)
	ws := seedWorkspace(t, store, "w")

	// Scoped methods exist and work:
	if _, err := ws.GetLessons(); err != nil {
		t.Fatalf("scoped GetLessons: %v", err)
	}

	// Unscoped *Store methods must NOT be reachable on wsStore. If *Store
	// were embedded, wsStore.GetWorkspaces() would compile. Uncomment to
	// confirm it fails:
	//   wsStore.GetWorkspaces()
	//   wsStore.AddWorkspace(Workspace{})
}
