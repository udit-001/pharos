package db

import (
	"context"
	"testing"
	"time"
)

// TestOpenPoolSurvivesOneLeakedConnection is the regression test for the
// "server starts but never responds" deadlock.
//
// Root cause: Open previously set SetMaxOpenConns(1). With a single
// connection, ANY code path that checks out a connection and fails to
// return it (a *sql.Rows whose Close was skipped, an uncommitted tx, a
// goroutine that died mid-query) permanently wedges the pool — every later
// query blocks forever on pool acquisition. busy_timeout cannot help: it
// governs the SQLite file lock, not Go's pool checkout. The dashboard
// boots, the listener accepts connections, and every DB-touching route
// hangs with zero bytes returned.
//
// This test models the exact failure: it leaks ONE connection (the common
// case — a single unclosed Rows) and asserts a normal store query still
// completes within a deadline. Under SetMaxOpenConns(1) GetWorkspaces
// deadlocks until the test timeout; with a pool of >=2 it returns promptly.
func TestOpenPoolSurvivesOneLeakedConnection(t *testing.T) {
	store := newTestStore(t)
	db := store.SQL()

	// Leak exactly one connection and hold it for the whole test, mimicking
	// a *sql.Rows that was never closed. Do NOT call Close until cleanup.
	leaked, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("checkout leak conn: %v", err)
	}
	t.Cleanup(func() { _ = leaked.Close() })

	// A normal store query must still complete. Under MaxOpenConns(1) this
	// blocks forever (the one conn is leaked); with a pool of >=2 it
	// returns fast on a different connection.
	done := make(chan error, 1)
	go func() {
		_, err := store.GetWorkspaces()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("GetWorkspaces after 1 leaked conn: %v", err)
		}
		// Success: the server keeps serving despite one leaked connection.
	case <-time.After(3 * time.Second):
		t.Fatalf("GetWorkspaces deadlocked after leaking 1 connection — " +
			"pool is sized so a single leak wedges everything (see maxOpenConns in db.go)")
	}
}

// TestOpenSetsSynchronousNormal asserts the WAL-recommended synchronous=NORMAL
// pragma is set on Open. NORMAL skips the per-commit fsync (fsync happens at
// checkpoint instead), shortening each write-lock hold — smaller collision
// window between agent CLI writes and human web writes on the shared file
// (LEARN-103). Values: 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA.
func TestOpenSetsSynchronousNormal(t *testing.T) {
	store := newTestStore(t)
	var sync int
	if err := store.SQL().QueryRow("PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if sync != 1 {
		t.Fatalf("PRAGMA synchronous = %d, want 1 (NORMAL)", sync)
	}
}
