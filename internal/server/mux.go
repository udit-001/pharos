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
	"github.com/udit-001/pharos/internal/docs"
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
	mux.HandleFunc("GET /about", handleAboutPage(store))
	mux.HandleFunc("GET /search", handleSearchPage(store))
	mux.HandleFunc("GET /api/lesson-html/{name}/{file}", handleLessonHTML(store))
	mux.HandleFunc("GET /api/ref-html/{name}/{file}", handleRefHTML(store))
	mux.HandleFunc("GET /api/lesson-html/{name}/assets/{file}", handleAssetFile(store))
	mux.HandleFunc("GET /api/ref-html/{name}/assets/{file}", handleAssetFile(store))

	return mux
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
	}
}

func toRenderWorkspace(ws db.Workspace, lessonCount, recordCount, refCount int) render.Workspace {
	return render.Workspace{
		Name:        ws.Name,
		Topic:       ws.Topic,
		LessonCount: lessonCount,
		RecordCount: recordCount,
		RefCount:    refCount,
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
		tl, tr, tRef := 0, 0, 0
		for _, w := range ws {
			tl += w.LessonCount
			tr += w.RecordCount
			tRef += w.RefCount
		}
		jsonResponse(w, map[string]any{
			"totalWorkspaces": len(ws), "totalLessons": tl, "totalRecords": tr, "totalRefs": tRef,
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
			var url string
			switch r.Type {
			case "lesson":
				url = urls.Lesson(r.WorkspaceName, r.SequenceNumber)
			case "record":
				url = urls.Record(r.WorkspaceName, r.SequenceNumber)
			case "ref":
				url = urls.Ref(r.WorkspaceName, r.Slug)
			}
			results = append(results, apiResult{
				Type: r.Type, Title: r.Title,
				URL: url, Summary: r.Summary, Snippet: r.Snippet, Workspace: r.WorkspaceName,
			})
		}
		jsonResponse(w, results)
	}
}

// ── Dashboard ──

func handleAboutPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writePage(w, nil, "About — Pharos", "", "", 0, "", "", render.About())
	}
}

func handleAppShell(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeNotFound(w, nil, "Page not found", "The page you're looking for doesn't exist.")
			return
		}

		ws, _ := store.GetWorkspaces()
		tl, tr, tRef := 0, 0, 0
		for _, w := range ws {
			tl += w.LessonCount
			tr += w.RecordCount
			tRef += w.RefCount
		}

		data := render.DashboardData{
			Stats: render.Stats{Workspaces: len(ws), Lessons: tl, Records: tr, Refs: tRef},
		}

		// Continue card
		for _, w := range ws {
			var continueURL, continueLabel string
			if w.LastLessonSeq != nil && *w.LastLessonSeq > 0 {
				wsStore, err := store.Workspace(w.Name)
				if err == nil {
					lessons, _ := wsStore.GetLessons()
					for _, l := range lessons {
						if l.SequenceNumber == *w.LastLessonSeq {
							continueURL = urls.Lesson(w.Name, l.SequenceNumber)
							continueLabel = fmt.Sprintf("%s — Lesson: %s", w.Name, l.Title)
							break
						}
					}
				}
			} else if w.LastRefSeq != nil {
				wsStore, err := store.Workspace(w.Name)
				if err == nil {
					refs, _ := wsStore.GetRefs()
					if len(refs) > 0 {
						ref := refs[0]
						continueURL = urls.Ref(w.Name, ref.Slug)
						continueLabel = fmt.Sprintf("%s — Reference: %s", w.Name, ref.Title)
					}
				}
			}
			if continueURL != "" {
				data.Continue = &render.ContinueItem{URL: continueURL, Label: continueLabel}
				break
			}
		}

		// Workspace grid
		for _, w := range ws {
			data.Workspaces = append(data.Workspaces, render.WorkspaceCard{
				Name: w.Name, Topic: w.Topic, LessonCount: w.LessonCount, RecordCount: w.RecordCount,
				RefCount: w.RefCount, LastStudied: w.LastStudied,
			})
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
			if trimmed := strings.TrimSpace(string(missionData)); !docs.IsTemplate(trimmed, "mission") {
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
			if !docs.IsTemplate(trimmed, kind) {
				// Strip a leading "# ..." H1 that duplicates the navbar title —
				// all document FORMAT templates start with one.
				if body := docs.StripH1(trimmed); body != "" {
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
		wsStore.SetLastViewed("ref", 0)
		sd, _ := wsStore.GetSidebarData()
		writePage(w, &sd, current.Title, name, "ref", 0, slug, "", render.Ref(data))
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
				var url string
				switch res.Type {
				case "lesson":
					url = urls.Lesson(res.WorkspaceName, res.SequenceNumber)
				case "record":
					url = urls.Record(res.WorkspaceName, res.SequenceNumber)
				case "ref":
					url = urls.Ref(res.WorkspaceName, res.Slug)
				}
				data.Results = append(data.Results, render.SearchResult{
					Type: res.Type, Title: res.Title,
					URL: url, Workspace: res.WorkspaceName,
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
