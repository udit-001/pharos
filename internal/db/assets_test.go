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

func TestInstallAsset_InstallIfAbsent(t *testing.T) {
	ws := newAssetWorkspace(t)
	spec := AssetSpec{
		Source:   "vendored",
		Filename: "mermaid.min.js",
		Files:    map[string][]byte{"mermaid-theme.js": []byte("theme")},
	}
	res, err := ws.InstallAsset(spec, []byte("LIB"), false)
	if err != nil {
		t.Fatalf("InstallAsset: %v", err)
	}
	if !res.LibWritten {
		t.Errorf("lib should be written, got %+v", res)
	}
	if len(res.FilesWritten) != 1 || res.FilesWritten[0] != "mermaid-theme.js" {
		t.Errorf("FilesWritten = %v, want [mermaid-theme.js]", res.FilesWritten)
	}
	// content on disk
	if got, _ := os.ReadFile(assetPath(ws, "mermaid.min.js")); string(got) != "LIB" {
		t.Errorf("lib content = %q", got)
	}
	if got, _ := os.ReadFile(assetPath(ws, "mermaid-theme.js")); string(got) != "theme" {
		t.Errorf("theme content = %q", got)
	}
}

func TestInstallAsset_SkipIfPresent(t *testing.T) {
	ws := newAssetWorkspace(t)
	if err := ws.WriteAsset("mermaid.min.js", []byte("old-lib")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteAsset("mermaid-theme.js", []byte("old-theme")); err != nil {
		t.Fatal(err)
	}
	spec := AssetSpec{
		Filename: "mermaid.min.js",
		Files:    map[string][]byte{"mermaid-theme.js": []byte("new-theme")},
	}
	res, err := ws.InstallAsset(spec, []byte("new-lib"), false)
	if err != nil {
		t.Fatalf("InstallAsset: %v", err)
	}
	if res.LibWritten {
		t.Errorf("lib should be skipped (present), was written")
	}
	if len(res.FilesWritten) != 0 {
		t.Errorf("files should be skipped, wrote %v", res.FilesWritten)
	}
	// content unchanged
	if got, _ := os.ReadFile(assetPath(ws, "mermaid-theme.js")); string(got) != "old-theme" {
		t.Errorf("theme overwritten: %q", got)
	}
	if got, _ := os.ReadFile(assetPath(ws, "mermaid.min.js")); string(got) != "old-lib" {
		t.Errorf("lib overwritten: %q", got)
	}
}

func TestInstallAsset_ForceOverwrites(t *testing.T) {
	ws := newAssetWorkspace(t)
	ws.WriteAsset("mermaid.min.js", []byte("old-lib"))
	ws.WriteAsset("mermaid-theme.js", []byte("old-theme"))
	spec := AssetSpec{
		Filename: "mermaid.min.js",
		Files:    map[string][]byte{"mermaid-theme.js": []byte("new-theme")},
	}
	res, err := ws.InstallAsset(spec, []byte("new-lib"), true)
	if err != nil {
		t.Fatalf("InstallAsset: %v", err)
	}
	if !res.LibWritten {
		t.Errorf("lib should be overwritten (force)")
	}
	if got, _ := os.ReadFile(assetPath(ws, "mermaid-theme.js")); string(got) != "new-theme" {
		t.Errorf("theme not overwritten: %q", got)
	}
	if got, _ := os.ReadFile(assetPath(ws, "mermaid.min.js")); string(got) != "new-lib" {
		t.Errorf("lib not overwritten: %q", got)
	}
}

// TestInstallAsset_ShortCircuitFix is the regression for the old asset_add
// bug: when the lib was present, the code returned early and never wrote
// companions — so a deleted mermaid-theme.js could not be restored via add.
// libData is nil (the CLI doesn't fetch when the lib is present); the
// companion is absent and must be written regardless.
func TestInstallAsset_ShortCircuitFix(t *testing.T) {
	ws := newAssetWorkspace(t)
	if err := ws.WriteAsset("mermaid.min.js", []byte("lib")); err != nil {
		t.Fatal(err)
	}
	// mermaid-theme.js deliberately absent
	spec := AssetSpec{
		Filename: "mermaid.min.js",
		Files:    map[string][]byte{"mermaid-theme.js": []byte("theme")},
	}
	res, err := ws.InstallAsset(spec, nil, false)
	if err != nil {
		t.Fatalf("InstallAsset: %v", err)
	}
	if res.LibWritten {
		t.Errorf("lib should be skipped (libData nil)")
	}
	if len(res.FilesWritten) != 1 || res.FilesWritten[0] != "mermaid-theme.js" {
		t.Errorf("companion not restored: %+v", res)
	}
	if got, _ := os.ReadFile(assetPath(ws, "mermaid-theme.js")); string(got) != "theme" {
		t.Errorf("companion content = %q, want theme", got)
	}
}

func TestInstallAsset_Seeded_NoLib(t *testing.T) {
	// Seeded asset: no Filename/URLTemplate, Files only, libData nil.
	ws := newAssetWorkspace(t)
	spec := AssetSpec{
		Source: "seeded",
		Files:  map[string][]byte{"glossary-tooltip.js": []byte("tooltip")},
	}
	res, err := ws.InstallAsset(spec, nil, false)
	if err != nil {
		t.Fatalf("InstallAsset: %v", err)
	}
	if res.LibWritten {
		t.Errorf("seeded asset should not write a lib")
	}
	if got, _ := os.ReadFile(assetPath(ws, "glossary-tooltip.js")); string(got) != "tooltip" {
		t.Errorf("seeded content = %q", got)
	}
}
