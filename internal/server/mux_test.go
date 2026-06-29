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
		{"lesson", "/workspace/alpha/lesson/1"},
		{"record", "/workspace/alpha/record/1"},
		{"ref", "/workspace/alpha/ref/ref-one"},
		{"quiz-library", "/workspace/alpha/quizzes"},
		{"search-page", "/search?q=Lesson"},
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

func TestLessonPagePrevNext(t *testing.T) {
	env := newTestEnv(t)

	// Lesson 1: should have "next" (Lesson Two) but no "prev"
	rec := env.get(t, "/workspace/alpha/lesson/1")
	body := rec.Body.String()
	if !strings.Contains(body, "Lesson One") {
		t.Error("lesson 1 page missing title 'Lesson One'")
	}
	if !strings.Contains(body, "Lesson Two") {
		t.Error("lesson 1 page should show next-link to 'Lesson Two'")
	}

	// Lesson 2: should have "prev" (Lesson One) but no "next"
	rec = env.get(t, "/workspace/alpha/lesson/2")
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

	env.get(t, "/workspace/alpha/lesson/2")

	// Verify SetLastViewed was called — workspace should redirect to lesson 2
	rec := env.get(t, "/workspace/alpha")
	if rec.Code != 302 {
		t.Errorf("workspace page should redirect after viewing a lesson; got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/lesson/2") {
		t.Errorf("redirect location = %q, want lesson/2", loc)
	}
}

func TestSearchPageResults(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/search?q=Lesson")
	body := rec.Body.String()
	if !strings.Contains(body, "Lesson One") {
		t.Error("search for 'Lesson' should find 'Lesson One'")
	}
	if !strings.Contains(body, "Lesson Two") {
		t.Error("search for 'Lesson' should find 'Lesson Two'")
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
	if !strings.Contains(body, "isn") {
		t.Error("404 should render styled not-found page")
	}
}

func TestDashboardContinueCard(t *testing.T) {
	env := newTestEnv(t)

	// View a lesson so LastLessonSeq is set
	env.get(t, "/workspace/alpha/lesson/1")

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

	rec := env.get(t, "/workspace/alpha/lesson/999")
	if rec.Code != 404 {
		t.Errorf("nonexistent lesson should 404; got %d", rec.Code)
	}
}

func TestQuizLibraryAndDetailPages(t *testing.T) {
	env := newTestEnv(t)
	wsStore, _ := env.store.Workspace("alpha")

	// Seed questions and a quiz containing them.
	if _, err := wsStore.AddQuestion(db.Question{
		Title: "Strongest ASD risk gene",
		Mode:  "choice",
		Config: `{"options":["CHD8","FMR1"],"key":0}`,
	}); err != nil {
		t.Fatalf("seed question: %v", err)
	}
	if _, err := wsStore.AddQuestion(db.Question{
		Title: "ASD heritability range",
		Mode:  "recall",
		Config: `{"reveal_text":"60-90%"}`,
	}); err != nil {
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
	})
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
