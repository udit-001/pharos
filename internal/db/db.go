package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/udit-001/pharos/internal/migrate"
	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database. The *sqlx.DB handle is private so that
// callers cannot bypass the typed query methods with raw SQL — the workspace
// scoping from WorkspaceStore stays enforced (LEARN-12).
type Store struct {
	db *sqlx.DB
}

// SQL exposes the underlying *sql.DB for migration tooling (goose). It is
// intentionally narrow: only the migrate package needs the raw handle.
func (s *Store) SQL() *sql.DB { return s.db.DB }

// Open opens (or creates) the SQLite database and runs migrations.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.DB.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Run goose migrations
	if err := migrate.Up(db.DB); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	store := &Store{db: db}
	_ = store.RebuildFTS()

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

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
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
	return s.db.Close()
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

func (s *Store) RebuildFTS() error {
	_, _ = s.db.Exec("INSERT INTO lessons_fts(lessons_fts) VALUES('rebuild')")
	_, _ = s.db.Exec("INSERT INTO records_fts(records_fts) VALUES('rebuild')")
	_, _ = s.db.Exec("INSERT INTO refs_fts(refs_fts) VALUES('rebuild')")
	return nil
}

var _ sql.DB
