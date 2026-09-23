package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/db"
)

// LEARN-235: `init --force` deletes pharos.db/-wal/-shm — the exact file-level
// mutation underneath a live connection that caused the 2026-09-24 corruption
// incident. While any Pharos process (server or CLI verb) holds the shared db
// lock, the recreate must refuse loudly instead of deleting the WAL.
func TestInitForceRefusesWhileDatabaseOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dataDir := t.TempDir()
	if err := config.Save(&config.Config{DataDir: dataDir}); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "pharos.db")

	// A live owner: this store holds the shared lock for its lifetime.
	s, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	root := newRootForTest()
	root.SetArgs([]string{"init", "--force"})
	err = root.Execute()
	if err == nil {
		t.Fatal("init --force must refuse while the database is open")
	}
	if !strings.Contains(err.Error(), "pharos stop") {
		t.Fatalf("refusal must name the fix (pharos stop), got: %v", err)
	}
	if _, statErr := os.Stat(dbPath); statErr != nil {
		t.Fatalf("database file must survive the refused recreate: %v", statErr)
	}
}

func TestInitForceSucceedsWhenNoHolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dataDir := t.TempDir()
	if err := config.Save(&config.Config{DataDir: dataDir}); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "pharos.db")

	s, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("initial open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Closed — no process holds the lock now.

	root := newRootForTest()
	root.SetArgs([]string{"init", "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init --force with no lock holder must succeed: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("recreated database missing: %v", err)
	}
}
