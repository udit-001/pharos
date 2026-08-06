package db

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSourceFor writes a temp source file outside the workspace, returns its path.
func writeSourceFor(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write source %s: %v", name, err)
	}
	return p
}

// seedSourceWS seeds a workspace with a real per-test temp dir path (NOT the
// shared /tmp/<name> that seedWorkspace uses), so on-disk sources/ files never
// collide across test functions or repeated runs.
func seedSourceWS(t *testing.T, store *Store, name string) *WorkspaceStore {
	t.Helper()
	if _, err := store.AddWorkspace(Workspace{Name: name, Path: t.TempDir()}); err != nil {
		t.Fatalf("seed source workspace %s: %v", name, err)
	}
	wsStore, err := store.Workspace(name)
	if err != nil {
		t.Fatalf("get source workspace %s: %v", name, err)
	}
	return wsStore
}

func TestCreateSourceDocRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "rt")

	body := "Chapter 1: Joins\nJoins combine rows.\n\nChapter 2: Indexes\nIndexes speed lookup.\n"
	src := writeSourceFor(t, "book.md", body)

	doc, err := ws.CreateSourceDoc(src, "SQL Joins")
	if err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}
	if doc.Title != "SQL Joins" {
		t.Errorf("title = %q", doc.Title)
	}
	if doc.Format != "markdown" {
		t.Errorf("format = %q, want markdown", doc.Format)
	}
	if doc.SHA256 == "" {
		t.Error("expected content hash")
	}
	if doc.EstimatedTokens == 0 {
		t.Error("expected token estimate")
	}

	// Raw file is kept in sources/.
	kept := filepath.Join(ws.Workspace().Path, "sources", doc.Filename)
	raw, err := os.ReadFile(kept)
	if err != nil {
		t.Fatalf("raw source not kept: %v", err)
	}
	if string(raw) != body {
		t.Errorf("raw source content mismatch")
	}

	// List + get by id.
	docs, err := ws.GetSourceDocs()
	if err != nil || len(docs) != 1 {
		t.Fatalf("GetSourceDocs = %d, err %v", len(docs), err)
	}
	got, err := ws.GetSourceDocByID(doc.ID)
	if err != nil {
		t.Fatalf("GetSourceDocByID: %v", err)
	}
	if got.Title != "SQL Joins" {
		t.Errorf("by id title = %q", got.Title)
	}
}

func TestCreateSourceDocIdempotentReingest(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "idem")
	src := writeSourceFor(t, "a.md", "# Title\nBody.\n")
	first, err := ws.CreateSourceDoc(src, "A")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Same content, same title → idempotent (returns existing, no new row).
	again, err := ws.CreateSourceDoc(src, "A")
	if err != nil {
		t.Fatalf("re-ingest same content: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("re-ingest created a new row: first=%d again=%d", first.ID, again.ID)
	}
	docs, err := ws.GetSourceDocs()
	if err != nil || len(docs) != 1 {
		t.Errorf("expected exactly 1 source doc after idempotent re-ingest, got %d (err %v)", len(docs), err)
	}
	// Different content under the same title is a conflict.
	other := writeSourceFor(t, "a.md", "# Title\nDifferent body.\n")
	if _, err := ws.CreateSourceDoc(other, "A"); err == nil {
		t.Error("expected conflict error for same-title different-content re-ingest")
	}
}

func TestSourceTextAndChapters(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "tc")
	src := writeSourceFor(t, "b.txt", "Chapter 1: One\nFirst body.\n\nChapter 2: Two\nSecond body.\n")
	doc, err := ws.CreateSourceDoc(src, "Book")
	if err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}

	text, err := ws.GetSourceText(doc.ID)
	if err != nil {
		t.Fatalf("GetSourceText: %v", err)
	}
	if !strings.Contains(text, "First body") || !strings.Contains(text, "Second body") {
		t.Errorf("text missing chapter bodies: %q", text)
	}

	ch, err := ws.GetSourceChapter(doc.ID, 2)
	if err != nil {
		t.Fatalf("GetSourceChapter: %v", err)
	}
	if !strings.Contains(ch.Segment, "Second body") {
		t.Errorf("chapter 2 segment = %q", ch.Segment)
	}
	if _, err := ws.GetSourceChapter(doc.ID, 99); err == nil {
		t.Error("expected out-of-range chapter error")
	}
}

func TestQuerySourcesAgentOnly(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "qa")
	src := writeSourceFor(t, "c.txt", "Chapter 1: Photosynthesis\nChlorophyll absorbs light.\n\nChapter 2: Respiration\nMitochondria produce energy.\n")
	if _, err := ws.CreateSourceDoc(src, "Bio"); err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}

	hits, err := ws.QuerySources("mitochondria")
	if err != nil {
		t.Fatalf("QuerySources: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("QuerySources hits = %d, want 1", len(hits))
	}
	if !strings.Contains(strings.ToLower(hits[0].Excerpt), "mitochondria") {
		t.Errorf("excerpt missing term: %q", hits[0].Excerpt)
	}
	if hits[0].Chapter != "Chapter 2: Respiration" {
		t.Errorf("chapter provenance = %q", hits[0].Chapter)
	}

	// Empty query → no hits.
	none, err := ws.QuerySources("")
	if err != nil || len(none) != 0 {
		t.Errorf("empty query: %d, err %v", len(none), err)
	}
}

func TestQuerySourcesDoesNotMixIntoUserSearch(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "qm")
	// A lesson whose body matches, plus a source doc with distinct content.
	if _, err := ws.AddLesson(Lesson{Title: "Lesson Alpha", Filename: "0001-alpha.html", Path: "lessons/0001-alpha.html"}); err != nil {
		t.Fatalf("AddLesson: %v", err)
	}
	src := writeSourceFor(t, "secret.txt", "Only the agent should see this unique needle phrase.\n")
	if _, err := ws.CreateSourceDoc(src, "Secret Notes"); err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}

	res, err := ws.Search("unique needle phrase")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range res {
		if r.Type == "source" || r.Title == "Secret Notes" {
			t.Errorf("user search leaked a source document: %+v", r)
		}
	}
	if len(res) != 0 {
		t.Errorf("user search unexpectedly hit %d results", len(res))
	}

	// But the dedicated source-retrieval surface finds it.
	hits, err := ws.QuerySources("unique needle phrase")
	if err != nil || len(hits) != 1 {
		t.Errorf("QuerySources = %d, err %v", len(hits), err)
	}
}

func TestCreateSourceDocPDF(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "pdf")

	// Real PDF fixture, copied out of the extract package's testdata so the
	// raw-file copy path is exercised.
	src := filepath.Join(t.TempDir(), "sample.pdf")
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "extract", "testdata", "sample.pdf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ws.CreateSourceDoc(src, "Manual")
	if err != nil {
		t.Fatalf("CreateSourceDoc pdf: %v", err)
	}
	if doc.Format != "pdf" {
		t.Errorf("format = %q, want pdf", doc.Format)
	}
	if doc.Pages == 0 {
		t.Error("expected page count > 0")
	}
	text, err := ws.GetSourceText(doc.ID)
	if err != nil {
		t.Fatalf("GetSourceText: %v", err)
	}
	if !strings.Contains(text, "This is a heading") {
		t.Errorf("pdf text missing content: %q", text)
	}
}

func TestCreateSourceDocDOCX(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "docx")

	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "extract", "testdata", "sample.docx"))
	if err != nil {
		t.Fatalf("read docx fixture: %v", err)
	}
	src := filepath.Join(t.TempDir(), "sample.docx")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ws.CreateSourceDoc(src, "Docx Doc")
	if err != nil {
		t.Fatalf("CreateSourceDoc docx: %v", err)
	}
	if doc.Format != "docx" {
		t.Errorf("format = %q, want docx", doc.Format)
	}
	text, err := ws.GetSourceText(doc.ID)
	if err != nil {
		t.Fatalf("GetSourceText: %v", err)
	}
	if !strings.Contains(text, "beta") {
		t.Errorf("docx text missing table content: %q", text)
	}
}

func TestCreateSourceDocEPUB(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "epub")
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "extract", "testdata", "sample.epub"))
	if err != nil {
		t.Fatalf("read epub fixture: %v", err)
	}
	src := filepath.Join(t.TempDir(), "sample.epub")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := ws.CreateSourceDoc(src, "Epub Book")
	if err != nil {
		t.Fatalf("CreateSourceDoc epub: %v", err)
	}
	if doc.Format != "epub" {
		t.Errorf("format = %q, want epub", doc.Format)
	}
	text, err := ws.GetSourceText(doc.ID)
	if err != nil {
		t.Fatalf("GetSourceText: %v", err)
	}
	if !strings.Contains(text, "Mitochondria") {
		t.Errorf("epub text missing content: %q", text)
	}
}

func TestQuerySourcesStemAwareChapterAttribution(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "stem")
	src := writeSourceFor(t, "s.txt", "Chapter 1: Fitness\nHe runs fast every morning.\n\nChapter 2: Diet\nSugary diets spike insulin.\n")
	if _, err := ws.CreateSourceDoc(src, "Stem Book"); err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}
	// "running" porter-stems to "run" and matches the token "runs" — but the
	// literal string "running" is absent from the text, so a literal excerpt
	// lookup would lose the match position and mis-attribute the chapter.
	hits, err := ws.QuerySources("running")
	if err != nil {
		t.Fatalf("QuerySources: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("QuerySources('running') hits = %d, want 1", len(hits))
	}
	if hits[0].Chapter != "Chapter 1: Fitness" {
		t.Errorf("chapter attribution = %q, want 'Chapter 1: Fitness' (excerpt %q)", hits[0].Chapter, hits[0].Excerpt)
	}
	if !strings.Contains(strings.ToLower(hits[0].Excerpt), "runs") {
		t.Errorf("excerpt should contain the matched surface 'runs': %q", hits[0].Excerpt)
	}
}

func TestDeleteSourceDoc(t *testing.T) {
	store := newTestStore(t)
	ws := seedSourceWS(t, store, "del")
	src := writeSourceFor(t, "d.txt", "Content.\n")
	doc, err := ws.CreateSourceDoc(src, "D")
	if err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}
	if err := ws.DeleteSourceDoc(doc.ID); err != nil {
		t.Fatalf("DeleteSourceDoc: %v", err)
	}
	if _, err := ws.GetSourceDocByID(doc.ID); !errors.Is(err, ErrSourceDocNotFound) {
		t.Errorf("expected not-found after delete, got %v", err)
	}
	// deleting again is a no-op
	if err := ws.DeleteSourceDoc(doc.ID); err != nil {
		t.Errorf("double delete: %v", err)
	}
}
