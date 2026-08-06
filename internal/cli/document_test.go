package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/db"
)

// TestDocumentExtractQuery drives the document CLI end-to-end against an
// injected store: ingest a source document, pull it by chapter, then query it.
func TestDocumentExtractQuery(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.AddWorkspace(db.Workspace{Name: "bio", Path: t.TempDir()}); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	src := filepath.Join(t.TempDir(), "notes.md")
	body := "# Photosynthesis\n\nChlorophyll absorbs light.\n\n## Respiration\n\nMitochondria produce energy.\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ingest + retrieval handle (--json).
	out := runWithStore(t, []string{"document", "extract", src, "--title", "Bio Notes", "-w", "bio", "--json"}, store)
	if !strings.Contains(out, `"format": "markdown"`) {
		t.Errorf("extract --json missing format:\n%s", out)
	}

	// Query finds the passage with provenance (dedicated source retrieval).
	q := runWithStore(t, []string{"document", "query", "mitochondria", "-w", "bio", "--json"}, store)
	if !strings.Contains(q, `"excerpt"`) || !strings.Contains(q, "Mitochondria") {
		t.Errorf("query did not return provenanced excerpt:\n%s", q)
	}

	// Whole-text pull via --text.
	full := runWithStore(t, []string{"document", "extract", src, "-w", "bio", "--text"}, store)
	if !strings.Contains(full, "Chlorophyll") {
		t.Errorf("--text missing content:\n%s", full)
	}
}

// TestDocumentExtractHandleShowsChapters proves the extract --json handle
// surfaces the chapter list (count + titles) so the agent can pick a chapter
// to pull with --chapter N.
func TestDocumentExtractHandleShowsChapters(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	if _, err := store.AddWorkspace(db.Workspace{Name: "b", Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "book.txt")
	body := "Chapter 1: Joins\nJoins merge rows.\n\nChapter 2: Indexes\nIndexes speed lookup.\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runWithStore(t, []string{"document", "extract", src, "--title", "Book", "-w", "b", "--json"}, store)
	if !strings.Contains(out, `"chapterCount": 2`) {
		t.Errorf("handle missing chapterCount 2:\n%s", out)
	}
	if !strings.Contains(out, "Joins") || !strings.Contains(out, "Indexes") {
		t.Errorf("handle missing chapter titles:\n%s", out)
	}
}
