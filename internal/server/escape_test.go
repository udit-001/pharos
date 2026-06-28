package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/urls"
)

// TestWorkspaceNamePathUnsafe is the regression test for LEARN-51: a
// workspace whose name contains path-unsafe characters (/ ? # %) must
// produce a URL that routes correctly. Before the fix, urls.PathEscape was
// space-only — a "/" in the name split the path segment and the route 404'd.
// After the fix, urls.PathEscape delegates to url.PathEscape, which encodes
// the full reserved set, and Go's ServeMux does not unescape %2F during
// matching, so the segment stays intact.
func TestWorkspaceNamePathUnsafe(t *testing.T) {
	for _, name := range []string{"bad/name", "bad?name", "bad#name", "bad%name"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			store, err := db.Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })

			wsDir := filepath.Join(dir, "ws")
			os.MkdirAll(wsDir, 0755)
			store.AddWorkspace(db.Workspace{Name: name, Topic: name, Path: wsDir})

			mux := NewMux(store, false)
			urlStr := urls.Workspace(name)
			t.Logf("name=%q  url=%q", name, urlStr)

			req := httptest.NewRequest("GET", urlStr, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status %d (want 200)", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "not found") || strings.Contains(rec.Body.String(), "doesn't exist") {
				t.Errorf("workspace exists but URL returned not-found")
			}
		})
	}
}
