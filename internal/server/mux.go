package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/docutil"
	"github.com/udit-001/pharos/internal/markdown"
	"github.com/udit-001/pharos/internal/render"
	"github.com/udit-001/pharos/internal/urls"
	"github.com/udit-001/pharos/internal/web"
)

// NewMux builds the HTTP mux for the Pharos dashboard: CSS serving, JSON API,
// page handlers, and raw file serving. This is the testable internal seam —
// tests construct the mux and drive routes through httptest.NewRecorder
// without booting a real server.
//
// devCSS serves CSS from disk (no embed, no-cache) for `pharos dev`.
func NewMux(store *db.Store, devCSS bool) *http.ServeMux {
	mux := http.NewServeMux()

	// Serve Tailwind CSS. In dev mode (DevCSS) read web/app.css from disk
	// on each request so styling changes are live without a Go rebuild;
	// disable caching so the browser always fetches the freshest file.
	mux.HandleFunc("GET /css/app.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if devCSS {
			data, err := os.ReadFile("web/app.css")
			if err != nil {
				http.Error(w, "app.css not built — run `pharos dev` from the project root", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Write(data)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.CSS)
	})

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconICO)
	})
	mux.HandleFunc("GET /favicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconPNG)
	})
	mux.HandleFunc("GET /favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconSVG)
	})

	// JSON API
	mux.HandleFunc("GET /api/workspaces", jsonHandler(handleListWorkspaces(store)))
	mux.HandleFunc("GET /api/workspaces/{id}", jsonHandler(handleGetWorkspace(store)))
	mux.HandleFunc("GET /api/workspaces/{id}/lessons", jsonHandler(handleListLessons(store)))
	mux.HandleFunc("GET /api/workspaces/{id}/records", jsonHandler(handleListRecords(store)))
	mux.HandleFunc("GET /api/workspaces/{id}/refs", jsonHandler(handleListRefs(store)))
	mux.HandleFunc("GET /api/workspaces/{id}/glossary-terms", jsonHandler(handleGetGlossaryTerms(store)))
	mux.HandleFunc("GET /api/workspaces/name/{name}/glossary-terms", jsonHandler(handleGetGlossaryTermsByName(store)))
	mux.HandleFunc("GET /api/stats", jsonHandler(handleStats(store)))
	mux.HandleFunc("GET /api/search", jsonHandler(handleSearch(store)))

	// App pages (rendered with sidebar)
	mux.HandleFunc("GET /", handleAppShell(store))
	mux.HandleFunc("GET /workspace/{name}", handleWorkspacePage(store))
	mux.HandleFunc("GET /workspace/{name}/mission", handleDocPage(store, "mission"))
	mux.HandleFunc("GET /workspace/{name}/resources", handleDocPage(store, "resources"))
	mux.HandleFunc("GET /workspace/{name}/glossary", handleGlossaryPage(store))
	mux.HandleFunc("GET /workspace/{name}/notes", handleDocPage(store, "notes"))
	mux.HandleFunc("GET /workspace/{name}/lesson/{seq}", handleLessonPage(store))
	mux.HandleFunc("GET /workspace/{name}/record/{seq}", handleRecordPage(store))
	mux.HandleFunc("GET /workspace/{name}/ref/{slug}", handleRefPage(store))
	mux.HandleFunc("GET /workspace/{name}/quizzes", handleQuizLibraryPage(store))
	mux.HandleFunc("GET /workspace/{name}/quiz/{slug}", handleQuizPage(store))
	mux.HandleFunc("POST /workspace/{name}/quiz/{slug}/start", handleQuizStart(store))
	mux.HandleFunc("GET /workspace/{name}/quiz/{slug}/attempt/{attemptID}", handleQuizAttemptPage(store))
	mux.HandleFunc("GET /workspace/{name}/quiz/{slug}/review/{attemptID}", handleQuizReviewPage(store))
	mux.HandleFunc("GET /about", handleAboutPage(store))
	mux.HandleFunc("GET /search", handleSearchPage(store))
	mux.HandleFunc("GET /api/lesson-html/{name}/{file}", handleLessonHTML(store))
	mux.HandleFunc("GET /api/ref-html/{name}/{file}", handleRefHTML(store))
	mux.HandleFunc("GET /api/lesson-html/{name}/assets/{file}", handleAssetFile(store))
	mux.HandleFunc("GET /api/ref-html/{name}/assets/{file}", handleAssetFile(store))
	mux.HandleFunc("POST /api/attempt", jsonHandler(handleSubmitAttempt(store)))
	mux.HandleFunc("POST /api/quiz-attempt/{id}/complete", jsonHandler(handleCompleteQuizAttempt(store)))
	mux.HandleFunc("POST /api/quiz-attempt/{id}/abandon", jsonHandler(handleAbandonQuizAttempt(store)))

	return mux
}

// dateShort returns the YYYY-MM-DD prefix of a timestamp.
func dateShort(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// ── helpers ──

func jsonHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, v any) {
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// workspaceByID resolves a workspace ID from the URL path and returns a
// WorkspaceStore. Used by the JSON API handlers that key on {id}.
func workspaceByID(store *db.Store, r *http.Request) (*db.WorkspaceStore, error) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	ws, err := store.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("workspace %d not found", id)
	}
	return store.WorkspaceByID(ws.ID)
}

// toRenderSidebar converts db sidebar data to render's view-model types.
// nil input returns an empty sidebar (dashboard, search pages).
func toRenderSidebar(sd *db.SidebarData) render.Sidebar {
	if sd == nil {
		return render.Sidebar{}
	}
	ws := toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs))
	return render.Sidebar{
		Workspace: &ws,
		Lessons:   toRenderLessons(sd.Lessons),
		Records:   toRenderRecords(sd.Records),
		Refs:      toRenderRefs(sd.Refs),
		Quizzes:   toRenderQuizzes(sd.Quizzes),
	}
}

func toRenderWorkspace(ws db.Workspace, lessonCount, recordCount, refCount int) render.Workspace {
	return render.Workspace{
		Name:        ws.Name,
		Topic:       ws.Topic,
		LessonCount: lessonCount,
		RecordCount: recordCount,
		RefCount:    refCount,
		QuizCount:   ws.QuizCount,
	}
}

func toRenderLessons(ls []db.SidebarLesson) []render.LessonEntry {
	out := make([]render.LessonEntry, len(ls))
	for i, l := range ls {
		out[i] = render.LessonEntry{Seq: l.Seq, Title: l.Title}
	}
	return out
}

func toRenderRecords(rs []db.SidebarRecord) []render.RecordEntry {
	out := make([]render.RecordEntry, len(rs))
	for i, r := range rs {
		out[i] = render.RecordEntry{Seq: r.Seq, Title: r.Title, Status: r.Status, Summary: r.Summary}
	}
	return out
}

func toRenderRefs(refs []db.SidebarRef) []render.RefEntry {
	out := make([]render.RefEntry, len(refs))
	for i, ref := range refs {
		out[i] = render.RefEntry{Slug: ref.Slug, Title: ref.Title}
	}
	return out
}

func toRenderQuizzes(qs []db.SidebarQuiz) []render.QuizEntry {
	out := make([]render.QuizEntry, len(qs))
	for i, q := range qs {
		out[i] = render.QuizEntry{Slug: q.Slug, Title: q.Title}
	}
	return out
}

// writePage renders a full page and writes it to the response. If sd is
// non-nil, it's used for the sidebar; nil gives an empty sidebar (dashboard,
// search).
func writePage(w http.ResponseWriter, sd *db.SidebarData, title, activeWS, activeType string, activeSeq int, activeSlug string, searchQuery string, content string) {
	f := render.Frame{
		Title:       title,
		ActiveWS:    activeWS,
		ActiveType:  activeType,
		ActiveSeq:   activeSeq,
		ActiveSlug:  activeSlug,
		SearchQuery: searchQuery,
		Sidebar:     toRenderSidebar(sd),
	}
	fmt.Fprint(w, render.Page(f, content))
}

// writeNotFound renders a styled 404 page inside the app frame. Pass a
// non-nil sd to show the sidebar (item-not-found in a valid workspace);
// nil gives a standalone page (workspace-not-found, unknown route).
func writeNotFound(w http.ResponseWriter, sd *db.SidebarData, title, message string) {
	w.WriteHeader(http.StatusNotFound)
	writePage(w, sd, title, "", "", 0, "", "", render.NotFound(title, message))
}

// ── API Handlers ──

func handleListWorkspaces(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, _ := store.GetWorkspaces()
		if ws == nil {
			ws = []db.Workspace{}
		}
		jsonResponse(w, ws)
	}
}

func handleGetWorkspace(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		ws, err := store.GetWorkspace(id)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		jsonResponse(w, ws)
	}
}

func handleListLessons(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsStore, err := workspaceByID(store, r)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		ls, _ := wsStore.GetLessons()
		if ls == nil {
			ls = []db.Lesson{}
		}
		jsonResponse(w, ls)
	}
}

func handleListRecords(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsStore, err := workspaceByID(store, r)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		recs, _ := wsStore.GetRecords()
		if recs == nil {
			recs = []db.LearningRecord{}
		}
		jsonResponse(w, recs)
	}
}

func handleListRefs(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsStore, err := workspaceByID(store, r)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		refs, _ := wsStore.GetRefs()
		if refs == nil {
			refs = []db.Reference{}
		}
		jsonResponse(w, refs)
	}
}

func handleGetGlossaryTerms(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsStore, err := workspaceByID(store, r)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		terms, _ := wsStore.GetGlossaryTerms()
		if terms == nil {
			terms = []db.GlossaryTerm{}
		}
		jsonResponse(w, terms)
	}
}

func handleGetGlossaryTermsByName(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		terms, _ := wsStore.GetGlossaryTerms()
		if terms == nil {
			terms = []db.GlossaryTerm{}
		}
		jsonResponse(w, terms)
	}
}

func handleStats(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, _ := store.GetWorkspaces()
		t := db.Totals(ws)
		jsonResponse(w, map[string]any{
			"totalWorkspaces": t.Workspaces, "totalLessons": t.Lessons, "totalRecords": t.Records, "totalRefs": t.Refs,
		})
	}
}

func handleSearch(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			jsonError(w, "missing q", 400)
			return
		}

		dbResults, _ := store.Search(q)
		type apiResult struct {
			Type      string `json:"type"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Summary   string `json:"summary"`
			Snippet   string `json:"snippet,omitempty"`
			Workspace string `json:"workspace"`
		}
		results := make([]apiResult, 0, len(dbResults))
		for _, r := range dbResults {
			results = append(results, apiResult{
				Type: r.Type, Title: r.Title,
				URL: searchResultURL(r), Summary: r.Summary, Snippet: r.Snippet, Workspace: r.WorkspaceName,
			})
		}
		jsonResponse(w, results)
	}
}

// searchResultURL maps a db.SearchResult to its dashboard URL. The single site
// for the SearchResult to URL mapping — shared by the JSON API (handleSearch)
// and the HTML page (handleSearchPage).
func searchResultURL(r db.SearchResult) string {
	switch r.Type {
	case "lesson":
		return urls.Lesson(r.WorkspaceName, r.SequenceNumber)
	case "record":
		return urls.Record(r.WorkspaceName, r.SequenceNumber)
	case "ref":
		return urls.Ref(r.WorkspaceName, r.Slug)
	case "quiz":
		return urls.Quiz(r.WorkspaceName, r.Slug)
	}
	return ""
}

// ── Dashboard ──

func handleAboutPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writePage(w, nil, "About", "", "", 0, "", "", render.About())
	}
}

func handleAppShell(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeNotFound(w, nil, "Page not found", "The page you're looking for doesn't exist.")
			return
		}

		ws, _ := store.GetWorkspaces()
		totals := db.Totals(ws)

		data := render.DashboardData{
			Stats: render.Stats{Workspaces: totals.Workspaces, Lessons: totals.Lessons, Records: totals.Records, Refs: totals.Refs, Quizzes: totals.Quizzes},
		}

		// Continue card
		if ci, _ := store.ContinueItem(); ci != nil {
			data.Continue = &render.ContinueItem{URL: ci.URL, Label: ci.Label}
		}

		// Workspace grid
		for _, w := range ws {
			data.Workspaces = append(data.Workspaces, render.WorkspaceCard{
				Name: w.Name, Topic: w.Topic, LessonCount: w.LessonCount, RecordCount: w.RecordCount,
				RefCount: w.RefCount, QuizCount: w.QuizCount, LastStudied: w.LastStudied,
			})
		}

		// Quiz widget — recent completed + in-progress.
		if qd, _ := store.GetQuizDashboardData(); qd.RecentCompleted != nil || len(qd.InProgress) > 0 {
			widget := &render.QuizWidgetData{}
			if qd.RecentCompleted != nil {
				widget.RecentCompleted = &render.QuizWidgetItem{
					WorkspaceName: qd.RecentCompleted.WorkspaceName,
					QuizTitle:     qd.RecentCompleted.QuizTitle,
					URL:           urls.QuizReview(qd.RecentCompleted.WorkspaceName, qd.RecentCompleted.QuizSlug, qd.RecentCompleted.AttemptID),
					Score:         qd.RecentCompleted.Score,
					Total:         qd.RecentCompleted.Total,
				}
			}
			for _, ip := range qd.InProgress {
				widget.InProgress = append(widget.InProgress, render.QuizWidgetItem{
					WorkspaceName: ip.WorkspaceName,
					QuizTitle:     ip.QuizTitle,
					URL:           urls.QuizAttemptPage(ip.WorkspaceName, ip.QuizSlug, ip.AttemptID),
				})
			}
			data.QuizWidget = widget
		}

		writePage(w, nil, "Dashboard", "", "", 0, "", "", render.Dashboard(data))
	}
}

// ── Workspace Page ──

func handleWorkspacePage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		ws := wsStore.Workspace()

		if ws.LastLessonSeq != nil {
			http.Redirect(w, r, urls.Lesson(name, *ws.LastLessonSeq), http.StatusFound)
			return
		}

		sd, err := wsStore.GetSidebarData()
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", "This workspace could not be loaded.")
			return
		}

		// Read mission from disk — the file is the source of truth,
		// not the DB column (which can go stale when the CLI writes
		// via --body-file or --edit without syncing back).
		// A mission with unresolved placeholders ({...}) counts as empty
		// — the workspace create command pre-populates the template.
		mission := ""
		if missionData, err := os.ReadFile(wsStore.Layout().MissionPath()); err == nil {
			if trimmed := strings.TrimSpace(string(missionData)); !docutil.IsTemplate(trimmed, "mission") {
				mission = trimmed
			}
		}

		// Render mission markdown → HTML (same pattern as learning records)
		missionHTML := ""
		if mission != "" {
			missionHTML = markdown.Render(mission)
		}

		data := render.WorkspaceData{
			Workspace: toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs)),
			Mission:   missionHTML,
			Lessons:   toRenderLessons(sd.Lessons),
			Records:   toRenderRecords(sd.Records),
			Refs:      toRenderRefs(sd.Refs),
		}
		writePage(w, &sd, ws.DisplayName(), ws.Name, "", 0, "", "", render.WorkspacePage(data))
	}
}

// ── Workspace Document Page (Mission, Resources, Glossary, Notes) ──

// docKind describes one workspace document readable from disk.
type docKind struct {
	title string
	path  func(db.Layout) string
}

var docKinds = map[string]docKind{
	"mission":   {title: "Mission", path: db.Layout.MissionPath},
	"resources": {title: "Resources", path: db.Layout.ResourcesPath},
	"notes":     {title: "Notes", path: db.Layout.NotesPath},
}

func handleDocPage(store *db.Store, kind string) http.HandlerFunc {
	dk, ok := docKinds[kind]
	if !ok {
		panic("unknown doc kind: " + kind)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		data := render.DocumentData{Title: dk.title, Kind: kind}

		raw, err := os.ReadFile(dk.path(wsStore.Layout()))
		if err == nil {
			trimmed := strings.TrimSpace(string(raw))
			// Workspace documents are seeded with placeholder templates on
			// create ({...} markers, or default prose for Notes). Treat an
			// unfilled template as empty so the learner gets guidance.
			if !docutil.IsTemplate(trimmed, kind) {
				// Strip a leading "# ..." H1 that duplicates the navbar title —
				// all document FORMAT templates start with one.
				if body := docutil.StripH1(trimmed); body != "" {
					data.BodyHTML = markdown.Render(body)
				}
			}
		}
		wsStore.Touch()
		if data.BodyHTML == "" {
			data.Empty = true
		}

		sd, _ := wsStore.GetSidebarData()
		writePage(w, &sd, dk.title, name, kind, 0, "", "", render.Document(data))
	}
}

// ── Glossary Page (rendered from DB, not from file) ──

func handleGlossaryPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		data := render.DocumentData{Title: "Glossary", Kind: "glossary"}
		terms, err := wsStore.GetGlossaryTerms()
		if err == nil && len(terms) > 0 {
			gt := make([]render.GlossaryTermRow, len(terms))
			for i, t := range terms {
				gt[i] = render.GlossaryTermRow{Term: t.Term, Definition: t.Definition, Category: t.Category, Avoid: t.Avoid}
			}
			data.GlossaryTerms = gt
		}
		if len(data.GlossaryTerms) == 0 {
			data.Empty = true
		}

		wsStore.Touch()
		sd, _ := wsStore.GetSidebarData()
		writePage(w, &sd, "Glossary", name, "glossary", 0, "", "", render.Document(data))
	}
}

// ── Lesson Page ──

func handleLessonPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		seq, _ := strconv.Atoi(r.PathValue("seq"))

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		current, err := wsStore.GetLessonBySeq(seq)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Lesson not found", fmt.Sprintf("Lesson #%d doesn't exist in this workspace.", seq))
			return
		}

		sd, err := wsStore.GetSidebarData()
		if err != nil {
			writeNotFound(w, nil, "Lesson not found", "This lesson could not be loaded.")
			return
		}

		data := render.LessonData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/lesson-html/%s/%s", urls.PathEscape(name), urls.PathEscape(current.Filename)),
			Seq:    seq,
			Total:  len(sd.Lessons),
		}
		wsStore.SetLastViewed("lesson", seq)
		writePage(w, &sd, current.Title, name, "lesson", seq, "", "", render.Lesson(data))
	}
}

// ── Record Page (MD → HTML) ──

func handleRecordPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		seq, _ := strconv.Atoi(r.PathValue("seq"))

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		ws := wsStore.Workspace()

		current, err := wsStore.GetRecordBySeq(seq)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Record not found", fmt.Sprintf("Learning record #%d doesn't exist in this workspace.", seq))
			return
		}

		recPath := filepath.Join(ws.Path, "learning-records", current.Filename)
		mdData, err := os.ReadFile(recPath)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Record not found", "This learning record's file could not be read.")
			return
		}

		data := render.RecordData{Title: current.Title, Status: current.Status, BodyHTML: markdown.Render(string(mdData))}
		wsStore.SetLastViewed("record", seq)
		sd, _ := wsStore.GetSidebarData()
		writePage(w, &sd, current.Title, name, "record", seq, "", "", render.Record(data))
	}
}

// ── Reference View Page ──

func handleRefPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		current, err := wsStore.GetRefBySlug(slug)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Reference not found", fmt.Sprintf("Reference %q doesn't exist in this workspace.", slug))
			return
		}

		data := render.RefData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/ref-html/%s/%s", urls.PathEscape(name), urls.PathEscape(current.Filename)),
		}
		wsStore.SetLastViewed("ref", int(current.ID))
		sd, _ := wsStore.GetSidebarData()
		writePage(w, &sd, current.Title, name, "ref", 0, slug, "", render.Ref(data))
	}
}

// ── Quiz Pages ──

func handleQuizLibraryPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		ws := wsStore.Workspace()

		sd, err := wsStore.GetSidebarData()
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", "This workspace could not be loaded.")
			return
		}

		quizzes, err := wsStore.GetQuizzes()
		if err != nil {
			writeNotFound(w, nil, "Quizzes not found", "This workspace's quizzes could not be loaded.")
			return
		}

		// Best scores come from the store's single scoring source
		// (GetQuizScores); the in-progress resume link is dashboard-only.
		scored, _ := wsStore.GetQuizScores()
		scoreBySlug := make(map[string]db.QuizScore, len(scored))
		for _, s := range scored {
			scoreBySlug[s.Slug] = s
		}

		entries := make([]render.QuizEntry, len(quizzes))
		var inProgress *render.QuizResumeLink
		for i, q := range quizzes {
			items, _ := q.ParseItems()
			total := len(items)
			entry := render.QuizEntry{
				Slug:        q.Slug,
				Title:       q.Title,
				Description: q.Description,
				ItemCount:   total,
				BestScore:   -1,
			}
			if s, ok := scoreBySlug[q.Slug]; ok && s.Attempted {
				entry.BestScore = s.BestScore
				entry.BestTotal = s.BestTotal
			}
			entries[i] = entry
			if attempts, err := wsStore.GetQuizAttempts(q.ID); err == nil {
				for _, a := range attempts {
					if a.Status == "in_progress" && inProgress == nil {
						scoredAns, _ := wsStore.GetAttempts(a.ID)
						inProgress = &render.QuizResumeLink{
							AttemptID: a.ID,
							QuizSlug:  q.Slug,
							QuizTitle: q.Title,
							Scored:    len(scoredAns),
							Total:     total,
						}
					}
				}
			}
		}

		data := render.QuizLibraryData{
			Workspace:  toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs)),
			Quizzes:    entries,
			InProgress: inProgress,
		}
		writePage(w, &sd, "Quizzes", ws.Name, "quiz-library", 0, "", "", render.QuizLibrary(data))
	}
}

func handleQuizPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		current, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Quiz not found", fmt.Sprintf("Quiz %q doesn't exist in this workspace.", slug))
			return
		}

		items, _ := current.ParseItems()
		sd, _ := wsStore.GetSidebarData()

		// Find in-progress and past completed attempts for this quiz.
		var inProgress int64
		var allCompleted []render.QuizAttemptSummary
		if attempts, err := wsStore.GetQuizAttempts(current.ID); err == nil {
			for _, a := range attempts {
				if a.Status == "in_progress" && inProgress == 0 {
					inProgress = a.ID
					continue
				}
				if a.Status == "completed" {
					correct, total := wsStore.ScoreAttempt(a.ID)
					allCompleted = append(allCompleted, render.QuizAttemptSummary{
						ID:          a.ID,
						Score:       correct,
						Total:       total,
						CompletedAt: dateShort(a.CompletedAt),
					})
				}
			}
		}

		// Cap at 3 most recent; show "and N more" for the rest.
		var pastAttempts []render.QuizAttemptSummary
		if len(allCompleted) > 3 {
			pastAttempts = allCompleted[:3]
		} else {
			pastAttempts = allCompleted
		}

		data := render.QuizData{
			Workspace:         toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs)),
			Slug:              current.Slug,
			Title:             current.Title,
			Description:       current.Description,
			ItemCount:         len(items),
			InProgressAttempt: inProgress,
			PastAttempts:      pastAttempts,
			ExtraAttemptCount: len(allCompleted) - len(pastAttempts),
		}
		writePage(w, &sd, current.Title, name, "quiz", 0, slug, "", render.Quiz(data))
	}
}

// ── Quiz Attempt Flow ──

func handleQuizStart(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		quiz, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			writeNotFound(w, nil, "Quiz not found", fmt.Sprintf("Quiz %q doesn't exist.", slug))
			return
		}

		// Resume an existing in-progress attempt if one exists.
		if attempts, err := wsStore.GetQuizAttempts(quiz.ID); err == nil {
			for _, a := range attempts {
				if a.Status == "in_progress" {
					http.Redirect(w, r, urls.QuizAttemptPage(name, slug, a.ID), http.StatusSeeOther)
					return
				}
			}
		}

		// Otherwise create a new attempt.
		qa, err := wsStore.CreateQuizAttempt(quiz.ID)
		if err != nil {
			writeNotFound(w, nil, "Could not start", "Failed to create a quiz attempt.")
			return
		}
		http.Redirect(w, r, urls.QuizAttemptPage(name, slug, qa.ID), http.StatusSeeOther)
	}
}

func handleQuizAttemptPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")
		attemptID, _ := strconv.ParseInt(r.PathValue("attemptID"), 10, 64)

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		quiz, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			writeNotFound(w, nil, "Quiz not found", fmt.Sprintf("Quiz %q doesn't exist.", slug))
			return
		}
		qa, err := wsStore.GetQuizAttempt(attemptID)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Attempt not found", "This quiz attempt could not be loaded.")
			return
		}
		if qa.Status != "in_progress" {
			http.Redirect(w, r, urls.QuizReview(name, slug, attemptID), http.StatusSeeOther)
			return
		}

		// Resolve question slugs to full questions (without correct answers
		// for choice; with reveal text for recall).
		slugs, _ := quiz.ParseItems()
		var questions []render.AttemptQuestion
		for _, qslug := range slugs {
			q, err := wsStore.GetQuestionBySlug(qslug)
			if err != nil {
				continue
			}
			cfg, _ := q.ParseConfig()
			var opts []string
			var reveal string
			if cc, ok := cfg.(db.ChoiceConfig); ok {
				opts = cc.Options
			}
			if rc, ok := cfg.(db.RecallConfig); ok {
				reveal = rc.RevealText
			}
			questions = append(questions, render.AttemptQuestion{
				ID:      q.ID,
				Title:   q.Title,
				Mode:    q.Mode,
				Options: opts,
				Reveal:  reveal,
			})
		}

		// Mark already-answered questions (for resume).
		answeredIDs := map[int64]bool{}
		answeredResults := map[int64]bool{}
		if answered, err := wsStore.GetAttempts(attemptID); err == nil {
			for _, a := range answered {
				answeredIDs[a.QuestionID] = true
				if a.Correct != nil {
					answeredResults[a.QuestionID] = *a.Correct
				}
			}
		}

		sd, _ := wsStore.GetSidebarData()
		data := render.AttemptData{
			Workspace:       toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs)),
			QuizSlug:        slug,
			QuizTitle:       quiz.Title,
			AttemptID:       attemptID,
			Questions:       questions,
			AnsweredIDs:     answeredIDs,
			AnsweredResults: answeredResults,
		}
		writePage(w, &sd, quiz.Title, name, "quiz", 0, slug, "", render.QuizAttempt(data))
	}
}

func handleQuizReviewPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")
		attemptID, _ := strconv.ParseInt(r.PathValue("attemptID"), 10, 64)

		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, nil, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		quiz, err := wsStore.GetQuizBySlug(slug)
		if err != nil {
			writeNotFound(w, nil, "Quiz not found", fmt.Sprintf("Quiz %q doesn't exist.", slug))
			return
		}
		qa, err := wsStore.GetQuizAttempt(attemptID)
		if err != nil {
			sd, _ := wsStore.GetSidebarData()
			writeNotFound(w, &sd, "Attempt not found", "This quiz attempt could not be loaded.")
			return
		}
		// An in-progress attempt has no review yet — send the user back to it.
		if qa.Status == "in_progress" {
			http.Redirect(w, r, urls.QuizAttemptPage(name, slug, attemptID), http.StatusSeeOther)
			return
		}

		// Build review items from quiz items + submitted answers.
		quizSlugs, _ := quiz.ParseItems()
		attempts, _ := wsStore.GetAttempts(attemptID)
		attemptMap := map[int64]db.Attempt{}
		for _, a := range attempts {
			attemptMap[a.QuestionID] = a
		}

		var items []render.ReviewItem
		for _, qslug := range quizSlugs {
			q, err := wsStore.GetQuestionBySlug(qslug)
			if err != nil {
				continue
			}
			cfg, _ := q.ParseConfig()
			att, hasAtt := attemptMap[q.ID]

			ri := render.ReviewItem{
				QuestionID:    q.ID,
				QuestionTitle: q.Title,
				Mode:          q.Mode,
			}
			if hasAtt {
				ri.UserResponse = att.Response
				if att.Correct != nil {
					ri.IsCorrect = *att.Correct
				}
			}
			if cc, ok := cfg.(db.ChoiceConfig); ok {
				ri.Options = cc.Options
				ri.CorrectIndex = cc.Key
			}
			if rc, ok := cfg.(db.RecallConfig); ok {
				ri.RevealText = rc.RevealText
			}
			items = append(items, ri)
		}

		correct, total := wsStore.ScoreAttempt(attemptID)
		sd, _ := wsStore.GetSidebarData()
		data := render.QuizReviewData{
			Workspace: toRenderWorkspace(sd.Workspace, len(sd.Lessons), len(sd.Records), len(sd.Refs)),
			QuizSlug:  slug,
			QuizTitle: quiz.Title,
			AttemptID: attemptID,
			Score:     correct,
			Total:     total,
			Items:     items,
		}
		writePage(w, &sd, "Review — "+quiz.Title, name, "quiz", 0, slug, "", render.QuizReview(data))
	}
}

func handleSubmitAttempt(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			QuizAttemptID int64  `json:"quiz_attempt_id"`
			QuestionID    int64  `json:"question_id"`
			Response      string `json:"response"`
			LatencyMs     int    `json:"latency_ms"`
			ClientCorrect *bool  `json:"correct"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		wsStore, err := store.QuizAttemptWorkspace(body.QuizAttemptID)
		if err != nil {
			jsonError(w, "quiz attempt not found", http.StatusNotFound)
			return
		}

		att, err := wsStore.SubmitAttempt(body.QuizAttemptID, body.QuestionID, body.Response, body.LatencyMs, body.ClientCorrect)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Resolve the correct index for client-side feedback display.
		question, err := wsStore.GetQuestionByID(body.QuestionID)
		correctIndex := 0
		if err == nil {
			if cc, ok := mustParseConfig(question); ok {
				correctIndex = cc.Key
			}
		}

		correct := false
		if att.Correct != nil {
			correct = *att.Correct
		}
		jsonResponse(w, map[string]any{
			"correct":       correct,
			"correct_index": correctIndex,
		})
	}
}

func mustParseConfig(q *db.Question) (db.ChoiceConfig, bool) {
	cfg, err := q.ParseConfig()
	if err != nil {
		return db.ChoiceConfig{}, false
	}
	cc, ok := cfg.(db.ChoiceConfig)
	return cc, ok
}

func handleCompleteQuizAttempt(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		wsStore, err := store.QuizAttemptWorkspace(id)
		if err != nil {
			jsonError(w, "quiz attempt not found", http.StatusNotFound)
			return
		}
		if err := wsStore.CompleteQuizAttempt(id); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		correct, total := wsStore.ScoreAttempt(id)
		jsonResponse(w, map[string]any{"ok": true, "correct": correct, "total": total})
	}
}

func handleAbandonQuizAttempt(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		wsStore, err := store.QuizAttemptWorkspace(id)
		if err != nil {
			jsonError(w, "quiz attempt not found", http.StatusNotFound)
			return
		}
		if err := wsStore.AbandonQuizAttempt(id); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(w, map[string]any{"ok": true})
	}
}

// ── Search Page ──

func handleSearchPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		data := render.SearchData{Query: q}

		if q != "" {
			dbResults, _ := store.Search(q)
			for _, res := range dbResults {
				data.Results = append(data.Results, render.SearchResult{
					Type: res.Type, Title: res.Title,
					URL: searchResultURL(res), Workspace: res.WorkspaceName,
					Summary: res.Summary, Snippet: res.Snippet,
				})
			}
		}

		writePage(w, nil, "Search", "", "", 0, "", q, render.Search(data))
	}
}

// ── Raw lesson/reference file serving (for iframes) ──

func handleLessonHTML(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		file := r.PathValue("file")
		wsStore, err := store.Workspace(name)
		if err != nil {
			iframeNotFound(w, "workspace", name)
			return
		}
		ws := wsStore.Workspace()
		serveFileOr404(w, r, filepath.Join(ws.Path, "lessons", file), "lesson", file)
	}
}

func handleRefHTML(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		file := r.PathValue("file")
		wsStore, err := store.Workspace(name)
		if err != nil {
			iframeNotFound(w, "workspace", name)
			return
		}
		ws := wsStore.Workspace()
		serveFileOr404(w, r, filepath.Join(ws.Path, "reference", file), "reference", file)
	}
}

// serveFileOr404 serves a file from disk, or a styled 404 page if the file
// is missing. Renders inside the lesson/reference iframe — styles are inlined
// so it renders correctly regardless of the iframe's document root.
func serveFileOr404(w http.ResponseWriter, r *http.Request, path, kind, file string) {
	if _, err := os.Stat(path); err == nil {
		http.ServeFile(w, r, path)
		return
	}
	iframeNotFound(w, kind, file)
}

// iframeNotFound writes a styled 404 page sized to render inside an iframe.
func iframeNotFound(w http.ResponseWriter, kind, ident string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, render.IframeNotFound(kind, ident))
}

func handleAssetFile(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		file := r.PathValue("file")
		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws := wsStore.Workspace()
		http.ServeFile(w, r, filepath.Join(ws.Path, "assets", file))
	}
}
