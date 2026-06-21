package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

// newTestStore opens a fresh temp SQLite database with migrations applied.
// Returned cleanup closes the store and removes the temp file.
func newTestStore(t *testing.T) (*db.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return store, func() { _ = store.Close() }
}

// runWithStore executes the named cobra subcommand with the given store
// injected via context (the seam created in LEARN-9). Captures stdout.
func runWithStore(t *testing.T, args []string, store *db.Store) string {
	t.Helper()
	root := newRootForTest()
	ctx := context.WithValue(context.Background(), ctxStore{}, store)

	root.SetArgs(args)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	root.PersistentPostRunE = nil

	// Commands print via fmt.Println to os.Stdout — redirect it.
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	err := root.ExecuteContext(ctx)
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return buf.String()
}

// newRootForTest returns a fresh rootCmd instance. Because the package uses
// init()-registered globals, we reuse the package-level rootCmd but reset its
// args/output per call.
func newRootForTest() *cobra.Command {
	// Reset persistent args / output on the shared rootCmd.
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetContext(context.Background())
	return rootCmd
}

// TestWorkspaceListWithInjectedStore proves the seam: a CLI command can be
// driven end-to-end against an injected store, with no global state and no
// reliance on the user's ~/.pharos database.
func TestWorkspaceListWithInjectedStore(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Seed two workspaces
	for _, name := range []string{"alpha", "beta"} {
		if _, err := store.AddWorkspace(db.Workspace{Name: name, Path: "/tmp/" + name}); err != nil {
			t.Fatalf("seed workspace %s: %v", name, err)
		}
	}

	out := runWithStore(t, []string{"workspace", "list"}, store)

	if !strings.Contains(out, "alpha") {
		t.Errorf("expected output to contain 'alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected output to contain 'beta', got:\n%s", out)
	}
	if !strings.Contains(out, "2 workspace(s)") {
		t.Errorf("expected '2 workspace(s)' in output, got:\n%s", out)
	}
}

// TestLessonListWithInjectedStore exercises a workspace-scoped command and
// verifies the injected store flows through resolveWorkspace too.
func TestLessonListWithInjectedStore(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ws, err := store.AddWorkspace(db.Workspace{Name: "golang", Path: "/tmp/golang"})
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if _, err := store.AddLesson(db.Lesson{WorkspaceID: ws.ID, Title: "Goroutines", Filename: "0001.html"}); err != nil {
		t.Fatalf("seed lesson: %v", err)
	}

	out := runWithStore(t, []string{"lesson", "list", "-w", "golang"}, store)

	if !strings.Contains(out, "Goroutines") {
		t.Errorf("expected output to contain 'Goroutines', got:\n%s", out)
	}
}
