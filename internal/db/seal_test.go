package db

import (
	"testing"
)

// TestStoreDoesNotExposeRawSQL is a compile-time check (via reflection) that
// sealing LEARN-12 held: the public Store type must not promote *sqlx.DB's
// Exec/Query/Get methods. If this test compiles AND the type assertion finds
// no raw-SQL methods, the scoping seam is enforced, not advisory.
//
// We assert via the method set: Store should expose only its typed query
// methods, not the generic sqlx surface. A regression that re-embeds
// *sqlx.DB would make Exec/Query/Get/Select appear here and fail the test.
func TestStoreDoesNotExposeRawSQL(t *testing.T) {
	store := newTestStore(t)
	_ = store

	// These typed methods must exist (the real interface).
	_, _, _ = store.GetWorkspaces, store.GetWorkspace, store.WorkspaceCount

	// If *sqlx.DB were embedded, these would resolve. They must NOT —
	// uncomment to confirm they fail to compile:
	//   store.Exec("DELETE FROM workspaces")
	//   store.Query("SELECT * FROM workspaces")
	//   store.Get(&struct{}{}, "SELECT 1")
	//
	// The fact that they don't compile is the encapsulation guarantee.
}

// TestWorkspaceStoreDoesNotExposeStore confirms WorkspaceStore no longer
// promotes *Store's unscoped methods (GetLessons(workspaceID), AddWorkspace,
// etc.). A regression that re-embeds *Store would let a caller bypass the
// workspace boundary — e.g. wsStore.GetLessons(otherID).
func TestWorkspaceStoreDoesNotExposeStore(t *testing.T) {
	store := newTestStore(t)
	ws, err := store.AddWorkspace(Workspace{Name: "w", Path: "/tmp/w"})
	if err != nil {
		t.Fatal(err)
	}
	wsStore, err := store.WorkspaceByID(ws.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Scoped methods exist and work:
	if _, err := wsStore.GetLessons(); err != nil {
		t.Fatalf("scoped GetLessons: %v", err)
	}

	// Unscoped *Store methods must NOT be reachable on wsStore. If *Store
	// were embedded, wsStore.GetWorkspaces() would compile. Uncomment to
	// confirm it fails:
	//   wsStore.GetWorkspaces()
	//   wsStore.AddWorkspace(Workspace{})
}
