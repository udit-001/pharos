package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/jmoiron/sqlx"
	"github.com/udit-001/pharos/internal/migrate"
	_ "modernc.org/sqlite"
)

// ErrDBLocked reports that another Pharos process holds the conflicting
// database lock — a shared holder (server or CLI verb) blocks destructive
// file operations, and an exclusive holder (init --force mid-recreate)
// blocks Open. Callers match on it to print the "stop the running server"
// fix instead of a raw syscall error.
var ErrDBLocked = errors.New("database is locked by another Pharos process")

// maxOpenConns caps the connection pool size.
//
// Do NOT set this to 1. With a single connection, any code path that checks
// out a connection and fails to return it (a leaked *sql.Rows whose Close is
// skipped, an uncommitted tx, a goroutine that died mid-query) permanently
// deadlocks the entire database: every subsequent query blocks forever on
// pool acquisition, and busy_timeout cannot help because it governs the
// SQLite file lock, not Go's pool checkout. The result is the dashboard
// booting, the listener accepting connections, and every DB-touching route
// hanging with zero bytes returned.
//
// A small pool (4) gives enough headroom that a single leak degrades
// performance instead of wedging the server, while staying low enough that
// SQLite writer contention is rare (WAL + busy_timeout=5000 serializes
// writers at the file level anyway). This is a mitigation; a leaked
// connection is still a bug to find and fix.
const maxOpenConns = 4

// Store wraps the SQLite database. The *sqlx.DB handle is private so that
// callers cannot bypass the typed query methods with raw SQL — the workspace
// scoping from WorkspaceStore stays enforced (LEARN-12).
//
// lock is the kernel-held shared advisory lock on <db>.lock (LEARN-235).
// It marks this process as a live owner of the database file: destructive
// operations (init --force) refuse to run while it is held, so a daemon's
// WAL can no longer be deleted out from under a live connection. Released
// in Close; the OS releases it on process death even on a crash.
type Store struct {
	db   *sqlx.DB
	lock *flock.Flock
}

// lockPath returns the advisory lock file for a database path. A separate
// lock FILE (not the db itself) avoids interfering with SQLite's own
// POSIX/byte-range locks and needs no sqlite cooperation: plain BSD flock
// on unix, LockFileEx on Windows, both via gofrs/flock.
func lockPath(dbPath string) string { return dbPath + ".lock" }

// acquireDBLock takes the shared (or exclusive, for destructive ops)
// non-blocking lock guarding the database file. ErrDBLocked means another
// Pharos process holds the conflicting mode.
func acquireDBLock(dbPath string, exclusive bool) (*flock.Flock, error) {
	l := flock.New(lockPath(dbPath))
	var locked bool
	var err error
	if exclusive {
		locked, err = l.TryLock()
	} else {
		locked, err = l.TryRLock()
	}
	if err != nil {
		return nil, fmt.Errorf("acquire db lock %s: %w", lockPath(dbPath), err)
	}
	if !locked {
		return nil, ErrDBLocked
	}
	return l, nil
}

// nowTimestamp returns the current UTC time as an RFC3339Nano string.
// This is the single source of truth for timestamp formatting across
// all write paths, ensuring ORDER BY works correctly on TEXT columns.
func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// SQL exposes the underlying *sql.DB for migration tooling (goose). It is
// intentionally narrow: only the migrate package needs the raw handle.
func (s *Store) SQL() *sql.DB { return s.db.DB }

// Open opens (or creates) the SQLite database and runs migrations.
//
// The caller must eventually Close the returned store — that releases the
// shared single-owner lock that marks this process as a live database
// owner (LEARN-235). Lock acquisition is non-blocking: if a destructive
// operation holds the exclusive lock, Open fails immediately with
// ErrDBLocked instead of racing it.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	lock, err := acquireDBLock(path, false)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.DB.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Snapshot DB before migrations so we can restore on failure.
	snapshotPath := path + ".pre-migrate"
	if err := snapshotDB(path, snapshotPath); err != nil {
		return nil, fmt.Errorf("snapshot db: %w", err)
	}

	// Run goose migrations
	if err := migrate.Up(db.DB); err != nil {
		restoreDB(path, snapshotPath)
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Migrations succeeded — clean up snapshot.
	os.Remove(snapshotPath)

	// Backfill slugs for lessons and learning records.
	// All operations are idempotent — safe to re-run on restart.
	if err := backfillSlugs(db.DB); err != nil {
		return nil, fmt.Errorf("backfill slugs: %w", err)
	}

	store := &Store{db: db, lock: lock}

	return store, nil
}

// OpenRaw opens a raw *sql.DB without migrations or sqlx wrapping.
// Used by the migrate CLI commands to avoid double-migration.
func OpenRaw(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	lock, err := acquireDBLock(path, false)
	if err != nil {
		return nil, err
	}
	// OpenRaw returns a bare *sql.DB with no Close hook we can attach the
	// lock to, and its callers are short-lived migrate verbs. Park the
	// handle in a registry so a GC finalizer cannot close the fd (and drop
	// the lock) while the verb runs; process exit releases the rest.
	rawLocks = append(rawLocks, lock)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func (s *Store) Close() error {
	defer func() {
		if s.lock != nil {
			_ = s.lock.Unlock()
		}
	}()
	return s.db.Close()
}

// rawLocks keeps OpenRaw's shared locks referenced for the life of the
// process. os.File-based locks are released when the fd is finalized, so a
// dropped reference would silently release the lock mid-verb. Migrate verbs
// are one-shot CLI processes; the kernel reclaims everything at exit.
var rawLocks []*flock.Flock

// AcquireExclusive takes the exclusive db lock for a destructive file
// operation (init --force's recreate). While any process — server or CLI
// verb — holds the shared lock, it fails with a message naming the fix
// instead of racing the live holder's WAL (LEARN-235). Callers release
// with Unlock before any subsequent shared acquisition (db.Open).
func AcquireExclusive(dbPath string) (*flock.Flock, error) {
	l, err := acquireDBLock(dbPath, true)
	if errors.Is(err, ErrDBLocked) {
		return nil, fmt.Errorf("another Pharos process has the database open — recreating it now would corrupt the data\n\n  Fix: run 'pharos stop' to stop the dashboard, then retry")
	}
	return l, err
}

// IndexSearch rebuilds the search index for all entity types across all
// workspaces by reading on-disk files and extracting plain text. Idempotent:
// already-indexed items are skipped.
func (s *Store) IndexSearch() (int, error) {
	wsList, err := s.GetWorkspaces()
	if err != nil {
		return 0, fmt.Errorf("list workspaces: %w", err)
	}
	var total int
	var errs []error
	for _, w := range wsList {
		wsStore, err := s.Workspace(w.Name)
		if err != nil {
			continue
		}
		if n, err := wsStore.IndexLessons(); err != nil {
			errs = append(errs, fmt.Errorf("workspace %q: %w", w.Name, err))
		} else {
			total += n
		}
		if n, err := wsStore.IndexRefs(); err != nil {
			errs = append(errs, fmt.Errorf("workspace %q: %w", w.Name, err))
		} else {
			total += n
		}
		if n, err := wsStore.IndexRecords(); err != nil {
			errs = append(errs, fmt.Errorf("workspace %q: %w", w.Name, err))
		} else {
			total += n
		}
	}
	return total, errors.Join(errs...)
}

// Search performs full-text search across all workspaces and all entity types
// (lessons, learning records, references). Returns flat results ordered by
// workspace (most recently studied first), then by type, then by sequence.
func (s *Store) Search(query string) ([]SearchResult, error) {
	wsList, err := s.GetWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	var results []SearchResult
	for _, w := range wsList {
		wsStore, err := s.Workspace(w.Name)
		if err != nil {
			continue
		}
		scoped, err := wsStore.Search(query)
		if err != nil {
			continue
		}
		results = append(results, scoped...)
	}
	if results == nil {
		return []SearchResult{}, nil
	}
	return results, nil
}

var _ sql.DB
