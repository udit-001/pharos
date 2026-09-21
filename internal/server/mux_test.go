package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/web"
)

// testEnv bundles a store and mux for server tests. The store uses a real
// SQLite temp file (matching db test pattern). Workspace files are written
// to a real temp dir so handlers that read from disk work correctly.
type testEnv struct {
	store *db.Store
	mux   *http.ServeMux
	wsDir string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	wsDir := filepath.Join(dir, "alpha")
	for _, sub := range []string{"lessons", "learning-records", "reference", "assets"} {
		os.MkdirAll(filepath.Join(wsDir, sub), 0755)
	}

	store.AddWorkspace(db.Workspace{Name: "alpha", Topic: "Alpha", Path: wsDir})
	wsStore, _ := store.Workspace("alpha")

	// Lessons (with files on disk for iframe serving)
	wsStore.AddLesson(db.Lesson{Title: "Lesson One", Filename: "0001-lesson-one.html", Path: "lessons/0001-lesson-one.html"})
	os.WriteFile(filepath.Join(wsDir, "lessons", "0001-lesson-one.html"), []byte("<h1>Lesson One</h1>"), 0644)
	wsStore.AddLesson(db.Lesson{Title: "Lesson Two", Filename: "0002-lesson-two.html", Path: "lessons/0002-lesson-two.html"})
	os.WriteFile(filepath.Join(wsDir, "lessons", "0002-lesson-two.html"), []byte("<h1>Lesson Two</h1>"), 0644)

	// Record (with .md on disk)
	wsStore.AddRecord(db.LearningRecord{Title: "Record One", Filename: "0001-record-one.md", Path: "learning-records/0001-record-one.md"})
	os.WriteFile(filepath.Join(wsDir, "learning-records", "0001-record-one.md"), []byte("# Record One\n\nSome learning."), 0644)

	// Reference
	wsStore.AddRef(db.Reference{Title: "Reference One", Slug: "ref-one", Filename: "ref-one.html", Path: "reference/ref-one.html"})
	os.WriteFile(filepath.Join(wsDir, "reference", "ref-one.html"), []byte("<h1>Ref One</h1>"), 0644)

	// Workspace documents — mission with real content, resources with placeholder
	os.WriteFile(filepath.Join(wsDir, "MISSION.md"), []byte("# Mission\n\nReal mission content"), 0644)
	os.WriteFile(filepath.Join(wsDir, "RESOURCES.md"), []byte("{some placeholder}"), 0644)
	os.WriteFile(filepath.Join(wsDir, "NOTES.md"), []byte("# Notes\n\nReal notes"), 0644)

	return &testEnv{store: store, mux: NewMux(store, false), wsDir: wsDir}
}

func (e *testEnv) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", target, nil)
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) post(t *testing.T, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) patch(t *testing.T, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) delete(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", target, nil)
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) workspaceID(t *testing.T) int64 {
	t.Helper()
	rec := e.get(t, "/api/workspaces")
	var wsList []db.Workspace
	json.Unmarshal(rec.Body.Bytes(), &wsList)
	if len(wsList) == 0 {
		t.Fatal("no workspaces in test store")
	}
	return wsList[0].ID
}

// ── Smoke tests: every route returns 200 + correct content-type ──

func TestSmokeAPIRoutes(t *testing.T) {
	env := newTestEnv(t)
	wsID := env.workspaceID(t)
	id := strconv.FormatInt(wsID, 10)

	cases := []struct {
		name        string
		path        string
		wantContent string
	}{
		{"workspaces", "/api/workspaces", "application/json"},
		{"workspace-by-id", "/api/workspaces/" + id, "application/json"},
		{"lessons", "/api/workspaces/" + id + "/lessons", "application/json"},
		{"records", "/api/workspaces/" + id + "/records", "application/json"},
		{"refs", "/api/workspaces/" + id + "/refs", "application/json"},
		{"stats", "/api/stats", "application/json"},
		{"glossary-terms", "/api/workspaces/" + id + "/glossary-terms", "application/json"},
		{"glossary-terms-by-name", "/api/workspaces/name/alpha/glossary-terms", "application/json"},
		{"search", "/api/search?q=Lesson", "application/json"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, c.wantContent) {
				t.Errorf("content-type = %q, want prefix %q", ct, c.wantContent)
			}
		})
	}
}

func TestHealthzProbe(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/healthz")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("cache-control = %q, want no-store (probe must never be served from cache)", cc)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"ok":true}` {
		t.Errorf("body = %q, want {\"ok\":true}", body)
	}
}

func TestPresenceJSBundle(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/js/presence.js")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("content-type = %q, want application/javascript", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, "pharosWatchServer") {
		t.Errorf("body missing pharosWatchServer export")
	}
}

func TestSmokePageRoutes(t *testing.T) {
	env := newTestEnv(t)

	cases := []struct {
		name string
		path string
	}{
		{"css", "/css/app.css"},
		{"dashboard", "/"},
		{"workspace", "/workspace/alpha"},
		{"mission", "/workspace/alpha/mission"},
		{"resources", "/workspace/alpha/resources"},
		{"glossary", "/workspace/alpha/glossary"},
		{"notes", "/workspace/alpha/notes"},
		{"lesson", "/workspace/alpha/lesson/lesson-one"},
		{"record", "/workspace/alpha/record/1"},
		{"ref", "/workspace/alpha/ref/ref-one"},
		{"quiz-library", "/workspace/alpha/quizzes"},
		{"lesson-html", "/api/lesson-html/alpha/0001-lesson-one.html"},
		{"ref-html", "/api/ref-html/alpha/ref-one.html"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "text/") {
				t.Errorf("content-type = %q, want text/ prefix", ct)
			}
		})
	}
}

// ── Deep tests: high-logic routes ──

func TestPWAStaticRoutes(t *testing.T) {
	env := newTestEnv(t)
	cases := []struct {
		name, path, wantContent string
	}{
		{"manifest", "/manifest.webmanifest", "application/manifest+json"},
		{"service-worker", "/sw.js", "application/javascript"},
		{"stopped-page", "/stopped.html", "text/html"},
		{"icon-192", "/icon-192.png", "image/png"},
		{"icon-512", "/icon-512.png", "image/png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, c.wantContent) {
				t.Errorf("content-type = %q, want prefix %q", ct, c.wantContent)
			}
		})
	}
}

func TestPWAHeadTags(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/")
	body := rec.Body.String()

	checks := []struct{ name, want string }{
		{"manifest link", `rel="manifest" href="/manifest.webmanifest"`},
		{"sentinel meta", `<meta name="pharos-app" content="1">`},
		{"theme-color", `<meta name="theme-color" id="theme-color" content="#eceff4">`},
		{"apple-touch-icon", `rel="apple-touch-icon" href="/icon-192.png"`},
		{"sw registration", "navigator.serviceWorker.register('/sw.js')"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.want) {
			t.Errorf("dashboard HTML missing %s", c.name)
		}
	}
}

func TestLessonPagePrevNext(t *testing.T) {
	env := newTestEnv(t)

	// Lesson 1: should have "next" (Lesson Two) but no "prev"
	rec := env.get(t, "/workspace/alpha/lesson/lesson-one")
	body := rec.Body.String()
	if !strings.Contains(body, "Lesson One") {
		t.Error("lesson 1 page missing title 'Lesson One'")
	}
	if !strings.Contains(body, "Lesson Two") {
		t.Error("lesson 1 page should show next-link to 'Lesson Two'")
	}

	// Lesson 2: should have "prev" (Lesson One) but no "next"
	rec = env.get(t, "/workspace/alpha/lesson/lesson-two")
	body = rec.Body.String()
	if !strings.Contains(body, "Lesson Two") {
		t.Error("lesson 2 page missing title 'Lesson Two'")
	}
	if !strings.Contains(body, "Lesson One") {
		t.Error("lesson 2 page should show prev-link to 'Lesson One'")
	}
}

func TestLessonPageSetsLastViewed(t *testing.T) {
	env := newTestEnv(t)

	env.get(t, "/workspace/alpha/lesson/lesson-two")

	// Workspace page should show a "Continue" card linking to the last-viewed lesson
	rec := env.get(t, "/workspace/alpha")
	if rec.Code != 200 {
		t.Fatalf("workspace page should render landing page; got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Continue: Lesson Two") {
		t.Error("workspace page should show continue card for last-viewed lesson")
	}
}

func TestSearchAPIResults(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/api/search?q=Lesson")
	var results []map[string]any
	json.Unmarshal(rec.Body.Bytes(), &results)
	if len(results) < 2 {
		t.Errorf("search for 'Lesson' returned %d results, want >= 2", len(results))
	}
}

// TestSearchAPIDoesNotReturnSources proves the boundary: an ingested source
// document must never appear in the user-facing /api/search.
func TestSearchAPIDoesNotReturnSources(t *testing.T) {
	env := newTestEnv(t)

	// Ingest a source document with content that also matches a user query.
	srcPath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(srcPath, []byte("Chapter 1: Photosynthesis\nChlorophyll absorbs light.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wsStore, err := env.store.Workspace("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wsStore.CreateSourceDoc(srcPath, "Botany Notes"); err != nil {
		t.Fatalf("CreateSourceDoc: %v", err)
	}

	// The source's unique term must not surface in user search, even though the
	// dedicated source-retrieval surface would find it.
	for _, q := range []string{"Chlorophyll", "Botany", "Photosynthesis"} {
		rec := env.get(t, "/api/search?q="+q)
		var results []map[string]any
		json.Unmarshal(rec.Body.Bytes(), &results)
		for _, r := range results {
			if r["title"] == "Botany Notes" {
				t.Errorf("source document leaked into /api/search for q=%q: %+v", q, r)
			}
		}
	}
}

func TestDocPagePlaceholderDetection(t *testing.T) {
	env := newTestEnv(t)

	// Mission has real content → should render
	rec := env.get(t, "/workspace/alpha/mission")
	body := rec.Body.String()
	if !strings.Contains(body, "Real mission content") {
		t.Error("mission page should render real content")
	}

	// Resources has {some placeholder} → should render empty state, not raw template
	rec = env.get(t, "/workspace/alpha/resources")
	body = rec.Body.String()
	if strings.Contains(body, "{some placeholder}") {
		t.Error("resources page should not render raw placeholder template content")
	}
}

func TestRecordPageRendersMarkdown(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/workspace/alpha/record/1")
	body := rec.Body.String()
	if !strings.Contains(body, "Record One") {
		t.Error("record page should contain title")
	}
	if !strings.Contains(body, "Some learning") {
		t.Error("record page should render markdown body content")
	}
}

func TestRefPageServesHTML(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/workspace/alpha/ref/ref-one")
	body := rec.Body.String()
	if !strings.Contains(body, "Reference One") {
		t.Error("ref page should contain title")
	}
}

func TestLessonHTMLNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/api/lesson-html/alpha/nonexistent.html")
	if rec.Code != 404 {
		t.Errorf("missing lesson HTML should 404; got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "can") {
		t.Error("404 should render styled not-found page")
	}
}

func TestGlossaryTooltipAutoInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Lesson WITH glossary-term — tooltip script should be injected.
	withGlossary := `<html><head></head><body><span class="glossary-term" data-term="JOIN">a join</span></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "glossary-lesson.html"), []byte(withGlossary), 0644)
	wsStore.AddLesson(db.Lesson{Title: "Glossary", Filename: "glossary-lesson.html"})

	rec := env.get(t, "/api/lesson-html/alpha/glossary-lesson.html")
	body := rec.Body.String()
	if !strings.Contains(body, "glossary-tooltip.js") {
		t.Error("lesson with glossary-term should auto-inject glossary-tooltip.js")
	}

	// Lesson WITHOUT glossary-term — tooltip script should NOT be injected.
	withoutGlossary := `<html><head></head><body><p>No glossary here</p></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "plain-lesson.html"), []byte(withoutGlossary), 0644)
	wsStore.AddLesson(db.Lesson{Title: "Plain", Filename: "plain-lesson.html"})

	rec = env.get(t, "/api/lesson-html/alpha/plain-lesson.html")
	body = rec.Body.String()
	if strings.Contains(body, "glossary-tooltip.js") {
		t.Error("lesson without glossary-term should NOT inject glossary-tooltip.js")
	}

	// Prose MENTIONING glossary-term (no such class) — no injection.
	prose := `<html><head></head><body><p>Wrap terms in a span with the glossary-term class.</p></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "prose-glossary.html"), []byte(prose), 0644)
	wsStore.AddLesson(db.Lesson{Title: "ProseGlossary", Filename: "prose-glossary.html"})

	rec = env.get(t, "/api/lesson-html/alpha/prose-glossary.html")
	if strings.Contains(rec.Body.String(), "glossary-tooltip.js") {
		t.Error("prose mentioning glossary-term must NOT inject glossary-tooltip.js (token match)")
	}
}

func TestCopyCodeAutoInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Lesson WITH data-copy — copy-code bundle should be injected.
	withCopy := `<html><head></head><body><pre data-copy><code>SELECT 1;</code></pre></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "copy-lesson.html"), []byte(withCopy), 0644)
	wsStore.AddLesson(db.Lesson{Title: "Copy", Filename: "copy-lesson.html"})

	rec := env.get(t, "/api/lesson-html/alpha/copy-lesson.html")
	body := rec.Body.String()
	if !strings.Contains(body, "copy-code.js") {
		t.Error("lesson with pre[data-copy] should auto-inject copy-code.js")
	}

	// Lesson WITHOUT data-copy — copy-code bundle should NOT be injected.
	withoutCopy := `<html><head></head><body><pre><code>SELECT 1;</code></pre></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "nocopy-lesson.html"), []byte(withoutCopy), 0644)
	wsStore.AddLesson(db.Lesson{Title: "NoCopy", Filename: "nocopy-lesson.html"})

	rec = env.get(t, "/api/lesson-html/alpha/nocopy-lesson.html")
	body = rec.Body.String()
	if strings.Contains(body, "copy-code.js") {
		t.Error("lesson without data-copy should NOT inject copy-code.js")
	}
}

func TestCopyCodeDetectionIsTokenBased(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Prose MENTIONING data-copy (no actual attribute) — no injection.
	// The old bytes.Contains check false-positived on this.
	prose := `<html><head></head><body><p>Mark snippets with the data-copy attribute.</p><pre><code>SELECT 1;</code></pre></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "prose-copy.html"), []byte(prose), 0644)
	wsStore.AddLesson(db.Lesson{Title: "ProseCopy", Filename: "prose-copy.html"})

	rec := env.get(t, "/api/lesson-html/alpha/prose-copy.html")
	if strings.Contains(rec.Body.String(), "copy-code.js") {
		t.Error("prose mentioning data-copy must NOT inject copy-code.js (token match, not substring)")
	}

	// A real pre[data-copy] attribute still injects.
	real := `<html><head></head><body><pre data-copy><code>SELECT 1;</code></pre></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "real-copy.html"), []byte(real), 0644)
	wsStore.AddLesson(db.Lesson{Title: "RealCopy", Filename: "real-copy.html"})

	rec = env.get(t, "/api/lesson-html/alpha/real-copy.html")
	if !strings.Contains(rec.Body.String(), "copy-code.js") {
		t.Error("pre[data-copy] should inject copy-code.js")
	}
}

func TestQuizBinderInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Lesson with .q quiz blocks — quiz binder injected.
	withQuiz := `<html><head></head><body><div class="q" data-answer="Bar chart"><p>Pick one</p><div class="options"><button>Bar chart</button></div><div class="fb"></div></div></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "quiz-lesson.html"), []byte(withQuiz), 0644)
	wsStore.AddLesson(db.Lesson{Title: "Quiz", Filename: "quiz-lesson.html"})

	rec := env.get(t, "/api/lesson-html/alpha/quiz-lesson.html")
	if !strings.Contains(rec.Body.String(), "pharos-quiz.js") {
		t.Error("lesson with .q blocks should auto-inject pharos-quiz.js")
	}

	// Lesson without .q — no quiz binder.
	withoutQuiz := `<html><head></head><body><p>No quiz here</p></body></html>`
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "noquiz-lesson.html"), []byte(withoutQuiz), 0644)
	wsStore.AddLesson(db.Lesson{Title: "NoQuiz", Filename: "noquiz-lesson.html"})

	rec = env.get(t, "/api/lesson-html/alpha/noquiz-lesson.html")
	if strings.Contains(rec.Body.String(), "pharos-quiz.js") {
		t.Error("lesson without .q should NOT inject pharos-quiz.js")
	}
}

func TestMermaidStackInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	mk := func(name, html string) {
		os.WriteFile(filepath.Join(env.wsDir, "lessons", name), []byte(html), 0644)
		wsStore.AddLesson(db.Lesson{Title: name, Filename: name})
	}
	lesson := `<html><head></head><body><div class="mermaid">graph TD; A-->B;</div></body></html>`

	// No vendored lib yet — no injection, no garbage tags.
	mk("mermaid-nolib.html", lesson)
	rec := env.get(t, "/api/lesson-html/alpha/mermaid-nolib.html")
	body := rec.Body.String()
	if strings.Contains(body, "mermaid.min.js") || strings.Contains(body, "pharos-mermaid.js") {
		t.Error(".mermaid without vendored lib must not inject the mermaid stack")
	}

	// After vendoring — full stack in order, glue last.
	os.WriteFile(filepath.Join(env.wsDir, "assets", "mermaid.min.js"), []byte("/* lib */"), 0644)
	os.WriteFile(filepath.Join(env.wsDir, "assets", "mermaid-theme.js"), []byte("/* theme */"), 0644)
	mk("mermaid-yes.html", lesson)

	rec = env.get(t, "/api/lesson-html/alpha/mermaid-yes.html")
	body = rec.Body.String()
	for _, want := range []string{"assets/mermaid.min.js", "assets/mermaid-theme.js", "/js/pharos-mermaid.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("mermaid stack missing %s", want)
		}
	}
	if strings.LastIndex(body, "assets/mermaid-theme.js") > strings.LastIndex(body, "pharos-mermaid.js") {
		t.Error("glue bundle must load after mermaid-theme.js")
	}

	// Lightbox vendored separately — included when present.
	os.WriteFile(filepath.Join(env.wsDir, "assets", "mermaid-lightbox.js"), []byte("/* lb */"), 0644)
	os.WriteFile(filepath.Join(env.wsDir, "assets", "mermaid-lightbox.css"), []byte("/* css */"), 0644)
	mk("mermaid-lb.html", lesson)
	rec = env.get(t, "/api/lesson-html/alpha/mermaid-lb.html")
	body = rec.Body.String()
	if !strings.Contains(body, "assets/mermaid-lightbox.js") || !strings.Contains(body, `href="assets/mermaid-lightbox.css"`) {
		t.Error("lightbox assets should join the mermaid stack when present")
	}

	// Non-mermaid lesson — nothing injected.
	mk("plain.html", `<html><head></head><body><p>plain</p></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/plain.html")
	if strings.Contains(rec.Body.String(), "pharos-mermaid.js") {
		t.Error("plain lesson should not inject mermaid glue")
	}
}

func TestKatexStackInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	vendKatex := func() {
		os.MkdirAll(filepath.Join(env.wsDir, "assets", "contrib"), 0755)
		os.WriteFile(filepath.Join(env.wsDir, "assets", "katex.min.js"), []byte("/* lib */"), 0644)
		os.WriteFile(filepath.Join(env.wsDir, "assets", "katex.min.css"), []byte("/* css */"), 0644)
		os.WriteFile(filepath.Join(env.wsDir, "assets", "katex-render.js"), []byte("/* render */"), 0644)
		os.WriteFile(filepath.Join(env.wsDir, "assets", "contrib", "auto-render.min.js"), []byte("/* ar */"), 0644)
	}
	mk := func(name, html string) {
		os.WriteFile(filepath.Join(env.wsDir, "lessons", name), []byte(html), 0644)
		wsStore.AddLesson(db.Lesson{Title: name, Filename: name})
	}

	// Currency in prose — no math, no injection (digit guard).
	money := `<html><head></head><body><p>The plan costs $5 per month, or $50 per year.</p></body></html>`
	vendKatex()
	mk("money.html", money)
	rec := env.get(t, "/api/lesson-html/alpha/money.html")
	if strings.Contains(rec.Body.String(), "katex") {
		t.Error("currency prose must not inject katex (digit guard)")
	}

	// Real inline math — full stack.
	math := `<html><head></head><body><p>The energy $E = mc^2$ is famous.</p></body></html>`
	mk("math.html", math)
	rec = env.get(t, "/api/lesson-html/alpha/math.html")
	body := rec.Body.String()
	for _, want := range []string{`href="assets/katex.min.css"`, "assets/katex.min.js", "assets/katex-render.js", "assets/contrib/auto-render.min.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("katex stack missing %s", want)
		}
	}

	// Unambiguous delimiters also fire.
	mk("delim.html", `<html><head></head><body><p>\(\alpha\) and $$\beta$$</p></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/delim.html")
	if !strings.Contains(rec.Body.String(), "assets/katex.min.js") {
		t.Error("\\( \\[ $$ delimiters should inject katex")
	}

	// Math inside pre/code is ignored (auto-render ignores those tags too).
	vendKatex()
	mk("code-money.html", `<html><head></head><body><pre data-copy><code>echo $$; # costs $5 and $10</code></pre></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/code-money.html")
	if strings.Contains(rec.Body.String(), "assets/katex.min.js") {
		t.Error("dollar signs inside pre/code must not inject katex")
	}
}

func TestVegaHighlightWorkbenchInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	mk := func(name, html string) {
		os.WriteFile(filepath.Join(env.wsDir, "lessons", name), []byte(html), 0644)
		wsStore.AddLesson(db.Lesson{Title: name, Filename: name})
	}
	vendoredFiles := func(names ...string) {
		for _, n := range names {
			os.WriteFile(filepath.Join(env.wsDir, "assets", n), []byte("/* "+n+" */"), 0644)
		}
	}

	// Vega: detected only with data-vega; stack gated on vega.min.js.
	mk("vega-nolib.html", `<html><head></head><body><div class="chart" data-vega="c1"></div></body></html>`)
	rec := env.get(t, "/api/lesson-html/alpha/vega-nolib.html")
	if strings.Contains(rec.Body.String(), "assets/vega") {
		t.Error("data-vega without vendored vega must not inject the stack")
	}

	vendoredFiles("vega.min.js", "vega-lite.min.js", "vega-embed.min.js", "vega-theme.js")
	mk("vega.html", `<html><head></head><body><div class="chart" data-vega="c1"></div><script type="application/json" id="c1">{}</script></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/vega.html")
	body := rec.Body.String()
	for _, want := range []string{"assets/vega.min.js", "assets/vega-lite.min.js", "assets/vega-embed.min.js", "assets/vega-theme.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("vega stack missing %s", want)
		}
	}
	if strings.Index(body, "assets/vega.min.js") > strings.Index(body, "assets/vega-theme.js") {
		// order check: vega.min.js must appear before vega-theme.js
		if strings.LastIndex(body, "assets/vega-theme.js") < strings.LastIndex(body, "assets/vega.min.js") {
			t.Error("vega-theme.js must load after vega.min.js")
		}
	}

	// Highlight: opt-in via language-* marker.
	mk("hl-no.html", `<html><head></head><body><pre><code>SELECT 1;</code></pre></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/hl-no.html")
	if strings.Contains(rec.Body.String(), "assets/highlight.min.js") || strings.Contains(rec.Body.String(), "pharos-hljs.js") {
		t.Error("plain pre>code must not inject highlight.js")
	}

	mk("hl-nolib.html", `<html><head></head><body><pre><code class="language-js">var x;</code></pre></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/hl-nolib.html")
	if strings.Contains(rec.Body.String(), "/js/pharos-hljs.js") {
		t.Error("language-* without vendored hljs must not inject")
	}

	vendoredFiles("highlight.min.js", "highlight.css")
	mk("hl.html", `<html><head></head><body><pre><code class="language-js">var x;</code></pre></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/hl.html")
	body = rec.Body.String()
	for _, want := range []string{`href="assets/highlight.css"`, "assets/highlight.min.js", "/js/pharos-hljs.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("hljs stack missing %s", want)
		}
	}

	// Workbench: custom element detection.
	mk("wb-nolib.html", `<html><head></head><body><sql-workbench namespace="l1" mode="card"></sql-workbench></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/wb-nolib.html")
	if strings.Contains(rec.Body.String(), "assets/sql-workbench.js") {
		t.Error("workbench element without vendored lib must not inject")
	}

	vendoredFiles("sql-workbench.js")
	mk("wb.html", `<html><head></head><body><sql-workbench namespace="l1" mode="card"></sql-workbench></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/wb.html")
	if !strings.Contains(rec.Body.String(), "assets/sql-workbench.js") {
		t.Error("<sql-workbench> should inject the workbench lib")
	}
}

func TestStyleAutoInjection(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// style.css is seeded into every real workspace at creation; the test
	// env skips the seed, so write it (its absence = degrade-skip path).
	os.WriteFile(filepath.Join(env.wsDir, "assets", "style.css"), []byte("/* base */"), 0644)

	mk := func(dir, name, html string) {
		os.WriteFile(filepath.Join(env.wsDir, dir, name), []byte(html), 0644)
		if dir == "lessons" {
			wsStore.AddLesson(db.Lesson{Title: name, Filename: name})
		} else {
			wsStore.AddRef(db.Reference{Title: name, Slug: name, Filename: name, Path: "reference/" + name})
		}
	}

	// Lesson without the stylesheet link — server injects it.
	mk("lessons", "no-style.html", `<html><head><title>T</title></head><body><p>lean</p></body></html>`)
	rec := env.get(t, "/api/lesson-html/alpha/no-style.html")
	if !strings.Contains(rec.Body.String(), `href="assets/style.css"`) {
		t.Error("lesson without style.css link should get it injected")
	}

	// Legacy lesson already linking it — no duplicate.
	mk("lessons", "has-style.html", `<html><head><link rel="stylesheet" href="assets/style.css"></head><body><p>legacy</p></body></html>`)
	rec = env.get(t, "/api/lesson-html/alpha/has-style.html")
	if strings.Count(rec.Body.String(), `href="assets/style.css"`) != 1 {
		t.Error("legacy lesson linking style.css must not get a duplicate")
	}

	// References get it too.
	mk("reference", "style-ref.html", `<html><head><title>R</title></head><body><p>ref</p></body></html>`)
	rec = env.get(t, "/api/ref-html/alpha/style-ref.html")
	if !strings.Contains(rec.Body.String(), `href="assets/style.css"`) {
		t.Error("reference without style.css link should get it injected")
	}

	// Question stimuli keep today's look — no stylesheet injection.
	os.MkdirAll(filepath.Join(env.wsDir, "questions"), 0755)
	os.WriteFile(filepath.Join(env.wsDir, "questions", "stim.html"), []byte(`<html><head></head><body><p>stimulus</p></body></html>`), 0644)
	rec = env.get(t, "/api/question-html/alpha/stim.html")
	if strings.Contains(rec.Body.String(), `href="assets/style.css"`) {
		t.Error("question stimuli must not get style.css injected")
	}

	// ...but feature injection DOES apply to stimuli (data-vis stimuli use
	// mermaid today).
	vend2 := map[string]string{"mermaid.min.js": "/* lib */", "mermaid-theme.js": "/* theme */"}
	for n, c := range vend2 {
		os.WriteFile(filepath.Join(env.wsDir, "assets", n), []byte(c), 0644)
	}
	os.WriteFile(filepath.Join(env.wsDir, "questions", "mermaid-stim.html"), []byte(`<html><head></head><body><div class="mermaid">graph TD; A-->B;</div></body></html>`), 0644)
	rec = env.get(t, "/api/question-html/alpha/mermaid-stim.html")
	if !strings.Contains(rec.Body.String(), "assets/mermaid.min.js") || !strings.Contains(rec.Body.String(), "pharos-mermaid.js") {
		t.Error("question stimulus with .mermaid should get the mermaid stack")
	}
}

func TestDashboardContinueCard(t *testing.T) {
	env := newTestEnv(t)

	// View a lesson so LastLessonSeq is set
	env.get(t, "/workspace/alpha/lesson/lesson-one")

	rec := env.get(t, "/")
	body := rec.Body.String()
	if !strings.Contains(body, "Lesson One") {
		t.Error("dashboard should show continue-card with last viewed lesson")
	}
}

func TestWorkspaceNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/workspace/nonexistent")
	if rec.Code != 404 {
		t.Errorf("nonexistent workspace should 404; got %d", rec.Code)
	}
}

func TestGlossaryTermsByNameAPI(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Seed some glossary terms
	if err := wsStore.AddGlossaryTerm("Hypertrophy", "Muscle growth from tension and stress", "", ""); err != nil {
		t.Fatalf("seed glossary term: %v", err)
	}
	if err := wsStore.AddGlossaryTerm("Progressive Overload", "Systematically increasing demand", "", ""); err != nil {
		t.Fatalf("seed glossary term: %v", err)
	}

	// Test name-based endpoint
	rec := env.get(t, "/api/workspaces/name/alpha/glossary-terms")
	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var terms []db.GlossaryTerm
	json.Unmarshal(rec.Body.Bytes(), &terms)
	if len(terms) != 2 {
		t.Errorf("got %d terms, want 2", len(terms))
	}
	if terms[0].Term != "Hypertrophy" {
		t.Errorf("first term = %q, want Hypertrophy", terms[0].Term)
	}

	// Test unknown workspace 404s
	rec = env.get(t, "/api/workspaces/name/nonexistent/glossary-terms")
	if rec.Code != 404 {
		t.Errorf("nonexistent workspace should 404; got %d", rec.Code)
	}
}

func TestLessonNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/workspace/alpha/lesson/nonexistent")
	if rec.Code != 404 {
		t.Errorf("nonexistent lesson should 404; got %d", rec.Code)
	}
}

func TestQuizLibraryAndDetailPages(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Seed questions and a quiz containing them.
	if _, err := wsStore.AddQuestion(db.Question{
		Title:  "Strongest ASD risk gene",
		Mode:   "choice",
		Config: `{"options":["CHD8","FMR1"],"key":0}`,
	}, ""); err != nil {
		t.Fatalf("seed question: %v", err)
	}
	if _, err := wsStore.AddQuestion(db.Question{
		Title:  "ASD heritability range",
		Mode:   "recall",
		Config: `{"reveal_text":"60-90%"}`,
	}, ""); err != nil {
		t.Fatalf("seed question: %v", err)
	}
	if _, err := wsStore.AddQuiz(db.Quiz{
		Title:       "Genetics foundations",
		Description: "Core genetic factors in ASD",
		Items:       `["strongest-asd-risk-gene","asd-heritability-range"]`,
	}); err != nil {
		t.Fatalf("seed quiz: %v", err)
	}

	// Library page lists the quiz.
	rec := env.get(t, "/workspace/alpha/quizzes")
	if rec.Code != 200 {
		t.Fatalf("quiz library status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Genetics foundations", "Core genetic factors in ASD", "2 questions"} {
		if !strings.Contains(body, want) {
			t.Errorf("quiz library missing %q", want)
		}
	}
	// Sidebar should show a Quizzes section.
	if !strings.Contains(body, `Quizzes</span>`) {
		t.Error("quiz library sidebar missing Quizzes section")
	}

	// Detail page shows title, description, item count, and Start button.
	rec = env.get(t, "/workspace/alpha/quiz/genetics-foundations")
	if rec.Code != 200 {
		t.Fatalf("quiz detail status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	for _, want := range []string{"Genetics foundations", "Core genetic factors in ASD", "2 questions", "Start quiz"} {
		if !strings.Contains(body, want) {
			t.Errorf("quiz detail missing %q", want)
		}
	}
	// Breadcrumb should carry the quiz title.
	if !strings.Contains(body, ">Genetics foundations<") {
		t.Error("quiz detail breadcrumb missing quiz title")
	}

	// Nonexistent quiz 404s.
	rec = env.get(t, "/workspace/alpha/quiz/no-such-quiz")
	if rec.Code != 404 {
		t.Errorf("nonexistent quiz should 404; got %d", rec.Code)
	}
}

func TestQuizAttemptAPIFlow(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Seed a choice question + quiz.
	q, err := wsStore.AddQuestion(db.Question{
		Title:  "Capital of France",
		Mode:   "choice",
		Config: `{"options":["London","Paris","Berlin"],"key":1}`,
	}, "")
	if err != nil {
		t.Fatalf("seed question: %v", err)
	}
	quiz, _ := wsStore.AddQuiz(db.Quiz{
		Title: "Geography",
		Items: `["` + q.Slug + `"]`,
	})
	_ = quiz

	// POST start → redirects to attempt page.
	rec := env.post(t, "/workspace/alpha/quiz/geography/start", "")
	if rec.Code != 303 {
		t.Fatalf("start status = %d, want 303; body: %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/attempt/") {
		t.Fatalf("redirect = %q, want /attempt/ path", loc)
	}

	// Extract attempt ID from the redirect.
	attemptID := strings.TrimPrefix(loc, "/workspace/alpha/quiz/geography/attempt/")

	// Attempt page renders.
	rec = env.get(t, loc)
	if rec.Code != 200 {
		t.Fatalf("attempt page status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "attempt-data") {
		t.Error("attempt page missing JSON data block")
	}
	if !strings.Contains(body, "Capital of France") {
		t.Error("attempt page missing question title")
	}

	// Submit correct answer via API.
	rec = env.post(t, "/api/attempt",
		`{"quiz_attempt_id":`+attemptID+`,"question_id":`+strconv.FormatInt(q.ID, 10)+`,"response":"1","latency_ms":2000}`)
	if rec.Code != 200 {
		t.Fatalf("submit status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"correct":true`) {
		t.Errorf("expected correct:true; got %s", rec.Body.String())
	}

	// Complete the attempt.
	rec = env.post(t, "/api/quiz-attempt/"+attemptID+"/complete", "")
	if rec.Code != 200 {
		t.Fatalf("complete status = %d; body: %s", rec.Code, rec.Body.String())
	}

	// Review page renders with score.
	rec = env.get(t, "/workspace/alpha/quiz/geography/review/"+attemptID)
	if rec.Code != 200 {
		t.Fatalf("review page status = %d", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "1/1") {
		t.Error("review page missing score 1/1")
	}

	// State machine: submit to completed attempt → error.
	rec = env.post(t, "/api/attempt",
		`{"quiz_attempt_id":`+attemptID+`,"question_id":`+strconv.FormatInt(q.ID, 10)+`,"response":"0","latency_ms":100}`)
	if rec.Code != 400 {
		t.Errorf("submit to completed attempt should 400; got %d", rec.Code)
	}
}

// ── Highlights API ──

func TestHighlightsAPI(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")
	lessons, _ := wsStore.GetLessons()
	lessonID := lessons[0].ID

	// GET empty.
	rec := env.get(t, "/api/workspaces/name/alpha/highlights?docType=lesson&docId="+strconv.FormatInt(lessonID, 10))
	if rec.Code != 200 {
		t.Fatalf("GET empty status = %d", rec.Code)
	}
	var empty []db.Highlight
	json.Unmarshal(rec.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected empty, got %d", len(empty))
	}

	// POST create.
	rec = env.post(t, "/api/workspaces/name/alpha/highlights",
		`{"docType":"lesson","docId":`+strconv.FormatInt(lessonID, 10)+`,"color":"#88C0D0","noteText":"key point","anchorData":"{\"text\":\"hello\",\"prefix\":\"a\",\"suffix\":\"b\"}"}`)
	if rec.Code != 200 {
		t.Fatalf("POST create status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var created db.Highlight
	json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("created ID not set")
	}
	if created.Color != "#88C0D0" {
		t.Errorf("color = %q, want #88C0D0", created.Color)
	}
	if created.NoteText != "key point" {
		t.Errorf("note = %q", created.NoteText)
	}
	if created.WorkspaceID != wsStore.Workspace().ID {
		t.Errorf("workspaceID not set from scope")
	}

	// GET returns created.
	rec = env.get(t, "/api/workspaces/name/alpha/highlights?docType=lesson&docId="+strconv.FormatInt(lessonID, 10))
	var list []db.Highlight
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].ID != created.ID {
		t.Errorf("GET after create: %+v", list)
	}

	// PATCH update.
	rec = env.patch(t, "/api/workspaces/name/alpha/highlights/"+strconv.FormatInt(created.ID, 10),
		`{"color":"#BF616A","noteText":"updated"}`)
	if rec.Code != 200 {
		t.Fatalf("PATCH status = %d", rec.Code)
	}

	// Verify update.
	rec = env.get(t, "/api/workspaces/name/alpha/highlights?docType=lesson&docId="+strconv.FormatInt(lessonID, 10))
	json.Unmarshal(rec.Body.Bytes(), &list)
	if list[0].Color != "#BF616A" {
		t.Errorf("updated color = %q, want #BF616A", list[0].Color)
	}
	if list[0].NoteText != "updated" {
		t.Errorf("updated note = %q", list[0].NoteText)
	}

	// DELETE.
	rec = env.delete(t, "/api/workspaces/name/alpha/highlights/"+strconv.FormatInt(created.ID, 10))
	if rec.Code != 204 {
		t.Errorf("DELETE status = %d, want 204", rec.Code)
	}

	// GET empty again.
	rec = env.get(t, "/api/workspaces/name/alpha/highlights?docType=lesson&docId="+strconv.FormatInt(lessonID, 10))
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("after delete: %d highlights, want 0", len(list))
	}

	// DELETE missing → 404.
	rec = env.delete(t, "/api/workspaces/name/alpha/highlights/"+strconv.FormatInt(created.ID, 10))
	if rec.Code != 404 {
		t.Errorf("delete missing status = %d, want 404", rec.Code)
	}

	// Unknown workspace → 404.
	rec = env.get(t, "/api/workspaces/name/nonexistent/highlights?docType=lesson&docId=1")
	if rec.Code != 404 {
		t.Errorf("unknown workspace status = %d, want 404", rec.Code)
	}
}

// TestHighlightIframeConfigInjection proves the lesson HTML iframe gets
// window.__pharos config + the highlights.js script injected.
func TestHighlightIframeConfigInjection(t *testing.T) {
	env := newTestEnv(t)

	// Overwrite with a proper HTML document (has </head>).
	properHTML := "<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Lesson One</h1><p>Some content</p></body></html>"
	os.WriteFile(filepath.Join(env.wsDir, "lessons", "0001-lesson-one.html"), []byte(properHTML), 0644)

	rec := env.get(t, "/api/lesson-html/alpha/0001-lesson-one.html")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "window.__pharos") {
		t.Error("iframe HTML missing window.__pharos config")
	}
	if !strings.Contains(body, "/js/pharos-highlights.js") {
		t.Error("iframe HTML missing pharos-highlights.js script")
	}
	if !strings.Contains(body, `"docType":"lesson"`) {
		t.Error("iframe HTML missing docType config")
	}

	// Ref HTML also gets highlights injection.
	os.WriteFile(filepath.Join(env.wsDir, "reference", "ref-one.html"), []byte("<!DOCTYPE html><html><head></head><body><h1>Ref</h1></body></html>"), 0644)
	rec = env.get(t, "/api/ref-html/alpha/ref-one.html")
	if rec.Code != 200 {
		t.Fatalf("ref status = %d", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, `"docType":"ref"`) {
		t.Error("ref iframe HTML missing docType config")
	}

	// Question HTML gets NO config (questions don't support highlights).
	os.MkdirAll(filepath.Join(env.wsDir, "questions"), 0755)
	os.WriteFile(filepath.Join(env.wsDir, "questions", "q1.html"), []byte("<!DOCTYPE html><html><head></head><body></body></html>"), 0644)
	rec = env.get(t, "/api/question-html/alpha/q1.html")
	if rec.Code != 200 {
		t.Fatalf("question status = %d", rec.Code)
	}
	body = rec.Body.String()
	if strings.Contains(body, "window.__pharos") {
		t.Error("question iframe should NOT have highlights config")
	}
}

func TestVersionedAssetURLsInFrame(t *testing.T) {
	env := newTestEnv(t)
	body := env.get(t, "/").Body.String()

	pres := web.JSBundleURL("presence.js")
	theme := web.JSBundleURL("pharos-theme.js")
	if pres == "" || theme == "" {
		t.Fatal("expected registered bundles resolved")
	}
	for _, want := range []string{
		`src="` + pres + `"`,
		`src="` + theme + `"`,
		`href="` + web.CSSURL() + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("frame body missing %s", want)
		}
	}
	for _, stale := range []string{
		`src="/js/presence.js"`, // must carry a version
		"?v=20", "?v=28",        // old manual counters must be gone
	} {
		if strings.Contains(body, stale) {
			t.Errorf("frame body still contains stale literal %q", stale)
		}
	}
}

func TestStoppedPageVersionedPresence(t *testing.T) {
	env := newTestEnv(t)
	body := env.get(t, "/stopped.html").Body.String()

	pres := web.JSBundleURL("presence.js")
	if pres == "" {
		t.Fatal("presence.js not registered")
	}
	if want := `src="` + pres + `"`; !strings.Contains(body, want) {
		t.Errorf("stopped page missing %s", want)
	}
	if strings.Contains(body, `src="/js/presence.js"`) {
		t.Error("stopped page references unversioned presence.js")
	}
}

func TestBundlesResolveVersioned(t *testing.T) {
	env := newTestEnv(t)
	for _, name := range []string{"presence.js", "pharos-theme.js", "pharos-toc.js", "glossary-tooltip.js"} {
		url := web.JSBundleURL(name)
		if url == "" {
			t.Errorf("%s not registered", name)
			continue
		}
		rec := env.get(t, url) // ?v= is a query — the route ignores it
		if rec.Code != 200 {
			t.Errorf("%s via %s: status = %d", name, url, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/javascript") {
			t.Errorf("%s: content-type = %q", name, ct)
		}
	}
}

// ── Workbench Handlers ──

func TestIngestWorkbenchEvents_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	body := `{"events":[{"id":"ev-1","namespace":"default","type":"query","ts":"2026-01-01T00:00:00Z","payload":{"sql":"SELECT 1","ok":true}}]}`
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", body)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var result map[string]int
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["accepted"] != 1 {
		t.Errorf("accepted = %d, want 1", result["accepted"])
	}
}

func TestIngestWorkbenchEvents_EmptyBatch(t *testing.T) {
	env := newTestEnv(t)
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", `{"events":[]}`)
	if rec.Code != 400 {
		t.Errorf("empty batch should 400; got %d", rec.Code)
	}
}

func TestIngestWorkbenchEvents_BatchOverCap(t *testing.T) {
	env := newTestEnv(t)

	// Build 51 events.
	events := make([]string, 51)
	for i := range events {
		events[i] = `{"id":"ev-` + itoa(i) + `","namespace":"default","type":"info","ts":"2026-01-01T00:00:00Z","payload":{}}`
	}
	body := `{"events":[` + strings.Join(events, ",") + `]}`
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", body)
	if rec.Code != 400 {
		t.Errorf("batch over cap should 400; got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestIngestWorkbenchEvents_ValidationFailure(t *testing.T) {
	env := newTestEnv(t)

	// Missing "type" field.
	body := `{"events":[{"id":"ev-1","ts":"2026-01-01T00:00:00Z","payload":{}}]}`
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", body)
	if rec.Code != 400 {
		t.Errorf("validation failure should 400; got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `event[0]`) {
		t.Errorf("error should mention event[0], got: %s", rec.Body.String())
	}
}

func TestIngestWorkbenchEvents_IdempotentRetry(t *testing.T) {
	env := newTestEnv(t)

	body := `{"events":[{"id":"ev-1","namespace":"default","type":"query","ts":"2026-01-01T00:00:00Z","payload":{"sql":"SELECT 1","ok":true}}]}`
	env.post(t, "/api/workspaces/name/alpha/workbench-events", body)

	// Retry same batch → accepted=0 (idempotent)
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", body)
	var result map[string]int
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["accepted"] != 0 {
		t.Errorf("retry accepted = %d, want 0", result["accepted"])
	}
}

func TestIngestWorkbenchEvents_UnknownWorkspace(t *testing.T) {
	env := newTestEnv(t)
	rec := env.post(t, "/api/workspaces/name/nonexistent/workbench-events",
		`{"events":[{"id":"ev-1","type":"info","ts":"2026-01-01T00:00:00Z","payload":{}}]}`)
	if rec.Code != 404 {
		t.Errorf("unknown workspace should 404; got %d", rec.Code)
	}
}

func TestIngestWorkbenchEvents_InvalidJSON(t *testing.T) {
	env := newTestEnv(t)
	rec := env.post(t, "/api/workspaces/name/alpha/workbench-events", `not json`)
	if rec.Code != 400 {
		t.Errorf("invalid JSON should 400; got %d", rec.Code)
	}
}

func TestGetDataset_HappyPath(t *testing.T) {
	env := newTestEnv(t)

	// Write a dataset file.
	datasetsDir := filepath.Join(env.wsDir, "datasets")
	os.MkdirAll(datasetsDir, 0755)
	os.WriteFile(filepath.Join(datasetsDir, "sample-data.json"), []byte(`{"rows":[1,2,3]}`), 0644)

	rec := env.get(t, "/api/workspaces/name/alpha/datasets/sample-data")
	if rec.Code != 200 {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	if !strings.Contains(rec.Body.String(), `"rows"`) {
		t.Error("dataset body missing expected content")
	}
}

func TestGetDataset_NotFound(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/api/workspaces/name/alpha/datasets/nonexistent")
	if rec.Code != 404 {
		t.Errorf("missing dataset should 404; got %d", rec.Code)
	}
}

func TestGetDataset_InvalidID(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/api/workspaces/name/alpha/datasets/UPPER-CASE")
	if rec.Code != 400 {
		t.Errorf("invalid dataset id should 400; got %d", rec.Code)
	}
}

func TestGetDataset_UnknownWorkspace(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/api/workspaces/name/nonexistent/datasets/sample")
	if rec.Code != 404 {
		t.Errorf("unknown workspace should 404; got %d", rec.Code)
	}
}
