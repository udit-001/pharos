package db

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/gofrs/flock"
)

// ── the single-owner contract (LEARN-235) ─────────────────────────────────
//
// Every process that opens the database — the server and each CLI verb —
// holds a SHARED kernel lock on pharos.db.lock for as long as its handle
// is open. Destructive file operations (init --force) must take an
// EXCLUSIVE lock and fail loudly while any shared holder exists. These
// tests pin that contract at the seam, without spawning real processes:
// two in-process flock handles on the same lock file conflict exactly the
// way two processes do.

// openLock is a test helper mirroring the lock path shape Open() uses.
func openLock(t *testing.T, dbPath string) *flock.Flock {
	t.Helper()
	return flock.New(lockPath(dbPath))
}

func TestOpenHoldsSharedLockForStoreLifetime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pharos.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// While the store is open, an exclusive lock must be refused.
	exclusive := openLock(t, path)
	locked, err := exclusive.TryLock()
	if err != nil {
		t.Fatalf("trylock: %v", err)
	}
	if locked {
		t.Fatal("exclusive lock taken while store holds a shared lock — single-owner contract violated")
	}
	_ = exclusive.Unlock()

	// After the store closes, the exclusive lock must be free.
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	locked, err = exclusive.TryLock()
	if err != nil {
		t.Fatalf("trylock after close: %v", err)
	}
	if !locked {
		t.Fatal("exclusive lock refused after store closed — lock leaked")
	}
	_ = exclusive.Unlock()
}

func TestOpenRefusesWhileDestructiveOpHoldsExclusive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pharos.db")

	exclusive := openLock(t, path)
	locked, err := exclusive.TryLock()
	if err != nil {
		t.Fatalf("trylock: %v", err)
	}
	if !locked {
		t.Fatal("precondition: exclusive lock must be acquirable on a fresh dir")
	}
	defer exclusive.Unlock()

	s, err := Open(path)
	if err == nil {
		_ = s.Close()
		t.Fatal("Open must refuse while a destructive op holds the exclusive lock")
	}
	if !errors.Is(err, ErrDBLocked) {
		t.Fatalf("err = %v, want ErrDBLocked", err)
	}
}

func TestOpenRawHoldsSharedLockForProcessLifetime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pharos.db")
	raw, err := OpenRaw(path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	defer raw.Close()

	// OpenRaw's lock is released at process exit, not on raw.Close — a
	// second raw handle must still see the lock held (the registry keeps
	// the handle alive against GC finalization).
	exclusive := openLock(t, path)
	locked, err := exclusive.TryLock()
	if err != nil {
		t.Fatalf("trylock: %v", err)
	}
	if locked {
		t.Fatal("exclusive lock taken while a raw handle holds a shared lock")
	}
	_ = exclusive.Unlock()

	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}
}
