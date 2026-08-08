package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

// The scratchpad (scrap/tag) is global — it does not need a workspace or
// filesystem. These tests drive the cobra commands against an injected store.

func writeBodyFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "body.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	return path
}

func TestScrapAddListReadUpdate(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Tags are strict — create them before attaching (tag create is the only
	// way; scratchpad never auto-creates tags).
	_ = runWithStore(t, []string{"tag", "create", "ml", "--description", "machine learning"}, store)
	_ = runWithStore(t, []string{"tag", "create", "career", "--description", "career goal"}, store)

	body := writeBodyFile(t, "roadmap: linear algebra first")
	out := runWithStore(t, []string{"scrap", "add", "ML engineer roadmap", "--body-file", body, "--tag", "ml", "--tag", "career"}, store)

	if !strings.Contains(out, "Scrap created") {
		t.Errorf("add output missing 'Scrap created':\n%s", out)
	}

	// Global — no workspace required. List active shows it.
	list := runWithStore(t, []string{"scrap", "list"}, store)
	if !strings.Contains(list, "ml-engineer-roadmap") {
		t.Errorf("list missing scrap:\n%s", list)
	}
	if !strings.Contains(list, "ml") || !strings.Contains(list, "career") {
		t.Errorf("list missing tags:\n%s", list)
	}

	// Read by slug.
	read := runWithStore(t, []string{"scrap", "read", "ml-engineer-roadmap"}, store)
	if !strings.Contains(read, "linear algebra first") {
		t.Errorf("read missing body:\n%s", read)
	}
}

func TestScrapListDefaultActiveAndStatusFilter(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	body := writeBodyFile(t, "b")
	b1 := runWithStore(t, []string{"scrap", "add", "active one", "--body-file", body}, store)
	_ = b1
	b2 := runWithStore(t, []string{"scrap", "add", "done one", "--body-file", body}, store)
	_ = b2
	_ = runWithStore(t, []string{"scrap", "update", "done-one", "--status", "done"}, store)

	active := runWithStore(t, []string{"scrap", "list"}, store)
	if strings.Contains(active, "done-one") {
		t.Errorf("default list should be ACTIVE only, found done-one:\n%s", active)
	}
	if !strings.Contains(active, "active-one") {
		t.Errorf("default list missing active-one:\n%s", active)
	}

	done := runWithStore(t, []string{"scrap", "list", "--status", "done"}, store)
	if !strings.Contains(done, "done-one") {
		t.Errorf("--status done missing done-one:\n%s", done)
	}
}

func TestScrapFindThenUpdateBySearch(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	body := writeBodyFile(t, "v1 content")
	_ = runWithStore(t, []string{"scrap", "add", "neural networks", "--body-file", body}, store)

	// Search finds it (find-then-update hook).
	found := runWithStore(t, []string{"scrap", "list", "--search", "networks"}, store)
	if !strings.Contains(found, "neural-networks") {
		t.Errorf("search missing scrap:\n%s", found)
	}

	// Update the same scrap by slug (body + title + tag), then verify it
	// updated (not duplicated) and the slug stayed stable.
	v2 := writeBodyFile(t, "v2 revised roadmap")
	_ = runWithStore(t, []string{"scrap", "update", "neural-networks", "--body-file", v2, "--title", "neural nets"}, store)
	only := runWithStore(t, []string{"scrap", "list"}, store)
	if strings.Count(only, "neural-networks") != 1 {
		t.Errorf("expected exactly one active scrap, got:\n%s", only)
	}
	read := runWithStore(t, []string{"scrap", "read", "neural-networks"}, store)
	if !strings.Contains(read, "v2 revised roadmap") {
		t.Errorf("read missing updated body:\n%s", read)
	}
	if !strings.Contains(read, "neural nets") {
		t.Errorf("read missing updated title:\n%s", read)
	}
	// Slug stable even though the title changed.
	if strings.Contains(read, "neural-nets\n") && !strings.Contains(read, "neural-networks") {
		t.Errorf("slug seems to have changed with title")
	}
}

func TestTagCreateListUpdateDelete(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_ = runWithStore(t, []string{"tag", "create", "ml", "--description", "machine learning career goal"}, store)

	list := runWithStore(t, []string{"tag", "list"}, store)
	if !strings.Contains(list, "ml") || !strings.Contains(list, "machine learning career goal") {
		t.Errorf("tag list missing name/description:\n%s", list)
	}

	_ = runWithStore(t, []string{"tag", "update", "ml", "--description", "revised goal"}, store)
	list = runWithStore(t, []string{"tag", "list"}, store)
	if !strings.Contains(list, "revised goal") {
		t.Errorf("tag list missing revised description:\n%s", list)
	}

	_ = runWithStore(t, []string{"tag", "delete", "ml"}, store)
	list = runWithStore(t, []string{"tag", "list"}, store)
	if strings.Contains(list, "ml") {
		t.Errorf("tag still present after delete:\n%s", list)
	}
}

func TestScrapInvalidStatusRejected(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	body := writeBodyFile(t, "b")
	_ = runWithStore(t, []string{"scrap", "add", "sample", "--body-file", body}, store)

	// Drive the update command and assert it returns an error (rather than
	// printing success) for an out-of-range status.
	root := newRootForTest()
	err := executeRootWithStore(root, []string{"scrap", "update", "sample", "--status", "bogus"}, store)
	if err == nil {
		t.Fatalf("expected error for invalid --status 'bogus', got nil")
	}
	if !strings.Contains(err.Error(), "active") || !strings.Contains(err.Error(), "done") {
		t.Errorf("error should mention valid statuses, got: %v", err)
	}
}

func executeRootWithStore(root *cobra.Command, args []string, store *db.Store) error {
	root.SetArgs(args)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	root.PersistentPostRunE = nil
	ctx := context.WithValue(context.Background(), ctxStore{}, store)
	return root.ExecuteContext(ctx)
}
