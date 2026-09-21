package db

import (
	"os"
	"path/filepath"
	"testing"
)

// newAssetWorkspace creates a workspace backed by a real temp dir (no seed),
// so asset methods have a clean assets/ directory to operate on.
func newAssetWorkspace(t *testing.T) *WorkspaceStore {
	t.Helper()
	store := newTestStore(t)
	dir := t.TempDir()
	if _, err := store.AddWorkspace(Workspace{Name: "assets-test", Path: dir}); err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	ws, err := store.Workspace("assets-test")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	return ws
}

func assetPath(ws *WorkspaceStore, name string) string {
	return filepath.Join(ws.ws.Path, "assets", name)
}

func TestWriteAsset_RoundtripAndSubdir(t *testing.T) {
	ws := newAssetWorkspace(t)
	if err := ws.WriteAsset("style.css", []byte("body{color:red}")); err != nil {
		t.Fatalf("WriteAsset: %v", err)
	}
	got, err := os.ReadFile(assetPath(ws, "style.css"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "body{color:red}" {
		t.Errorf("content = %q, want body{color:red}", got)
	}
	// subdir (fonts/x) allowed and parent created
	if err := ws.WriteAsset(filepath.Join("fonts", "inter-latin.woff2"), []byte("FONT")); err != nil {
		t.Fatalf("WriteAsset subdir: %v", err)
	}
	if _, err := os.Stat(assetPath(ws, filepath.Join("fonts", "inter-latin.woff2"))); err != nil {
		t.Errorf("subdir asset not written: %v", err)
	}
}

func TestWriteAsset_TraversalRejected(t *testing.T) {
	ws := newAssetWorkspace(t)
	for _, bad := range []string{"../../etc/passwd", "/etc/passwd", "", ".", "fonts/../.."} {
		if err := ws.WriteAsset(bad, []byte("x")); err == nil {
			t.Errorf("WriteAsset(%q) expected error, got nil", bad)
		}
	}
}

func TestDeleteAsset(t *testing.T) {
	ws := newAssetWorkspace(t)
	if err := ws.WriteAsset("quiz.js", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteAsset("quiz.js"); err != nil {
		t.Fatalf("DeleteAsset: %v", err)
	}
	names, err := ws.ListAssets()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if n == "quiz.js" {
			t.Errorf("quiz.js still present after delete: %v", names)
		}
	}
	// absent -> clear "not found" error
	if err := ws.DeleteAsset("missing.js"); err == nil {
		t.Errorf("DeleteAsset(missing) expected error, got nil")
	}
	// traversal -> error
	if err := ws.DeleteAsset("../../etc/passwd"); err == nil {
		t.Errorf("DeleteAsset(traversal) expected error, got nil")
	}
}
