package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/render"
	"github.com/udit-001/pharos/internal/web"
	"github.com/yuin/goldmark"
)

// ── globals shared across handlers ──

var md = goldmark.New()

// ── Config ──

type Config struct {
	Port   int
	DB     *db.Store
	NoOpen bool
	Silent bool
	DevCSS bool // serve CSS from disk (no embed, no-cache) for `pharos dev`
}

// ── Start ──

func Start(cfg Config) error {
	mux := http.NewServeMux()

	// Serve Tailwind CSS. In dev mode (DevCSS) read web/app.css from disk
	// on each request so styling changes are live without a Go rebuild;
	// disable caching so the browser always fetches the freshest file.
	mux.HandleFunc("GET /css/app.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if cfg.DevCSS {
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
	mux.HandleFunc("GET /api/workspaces", jsonHandler(handleListWorkspaces(cfg.DB)))
	mux.HandleFunc("GET /api/workspaces/{id}", jsonHandler(handleGetWorkspace(cfg.DB)))
	mux.HandleFunc("GET /api/workspaces/{id}/lessons", jsonHandler(handleListLessons(cfg.DB)))
	mux.HandleFunc("GET /api/workspaces/{id}/records", jsonHandler(handleListRecords(cfg.DB)))
	mux.HandleFunc("GET /api/workspaces/{id}/refs", jsonHandler(handleListRefs(cfg.DB)))
	mux.HandleFunc("GET /api/stats", jsonHandler(handleStats(cfg.DB)))
	mux.HandleFunc("GET /api/search", jsonHandler(handleSearch(cfg.DB)))

	// App pages (rendered with sidebar)
	mux.HandleFunc("GET /", handleAppShell(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}", handleWorkspacePage(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}/mission", handleDocPage(cfg.DB, "mission"))
	mux.HandleFunc("GET /workspace/{name}/resources", handleDocPage(cfg.DB, "resources"))
	mux.HandleFunc("GET /workspace/{name}/glossary", handleDocPage(cfg.DB, "glossary"))
	mux.HandleFunc("GET /workspace/{name}/notes", handleDocPage(cfg.DB, "notes"))
	mux.HandleFunc("GET /workspace/{name}/lesson/{seq}", handleLessonPage(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}/record/{seq}", handleRecordPage(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}/ref/{slug}", handleRefPage(cfg.DB))
	mux.HandleFunc("GET /search", handleSearchPage(cfg.DB))
	mux.HandleFunc("GET /api/lesson-html/{name}/{file}", handleLessonHTML(cfg.DB))
	mux.HandleFunc("GET /api/ref-html/{name}/{file}", handleRefHTML(cfg.DB))
	mux.HandleFunc("GET /api/lesson-html/{name}/assets/{file}", handleAssetFile(cfg.DB))
	mux.HandleFunc("GET /api/ref-html/{name}/assets/{file}", handleAssetFile(cfg.DB))

	// Try the configured port, then port+1, port+2, … up to 100 attempts.
	var listener net.Listener
	port := cfg.Port
	for i := 0; i < 100; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		var err error
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
		port++
	}
	if listener == nil {
		return fmt.Errorf("no free port found starting from %d", cfg.Port)
	}
	cfg.Port = port // store the actual port for messages below

	srv := &http.Server{Handler: mux}

	if !cfg.NoOpen && !cfg.Silent {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		if err := openBrowser(url); err != nil {
			log.Printf("  Open %s in your browser", url)
		}
	}
	if cfg.Silent {
		log.Printf("Pharos listening on http://127.0.0.1:%d", port)
	} else {
		fmt.Printf("  Pharos Dashboard: http://127.0.0.1:%d\n", port)
		fmt.Println()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; srv.Close() }()
	// http.ErrServerClosed is the expected return from a graceful SIGINT/SIGTERM
	// shutdown (srv.Close above) — treat it as a clean exit, not an error,
	// so `pharos start`/`pharos dev` don't print a spurious error on stop.
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ── helpers ──

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

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

// sidebarData loads the workspace tree for the sidebar. The workspace is nil
// when activeWS is empty (dashboard / search) — render shows an empty state.
func sidebarData(store *db.Store, activeWS string) render.Sidebar {
	if activeWS == "" {
		return render.Sidebar{}
	}
	wsStore, err := store.Workspace(activeWS)
	if err != nil {
		return render.Sidebar{}
	}
	ws := wsStore.Workspace()
	lessons, _ := wsStore.GetLessons()
	records, _ := wsStore.GetRecords()
	refs, _ := wsStore.GetRefs()
	return render.Sidebar{Workspace: &ws, Lessons: lessons, Records: records, Refs: refs}
}

// frame builds the render.Frame for a page.
func frame(store *db.Store, title, activeWS, activeType string, activeSeq int, activeSlug string) render.Frame {
	return render.Frame{
		Title:      title,
		ActiveWS:   activeWS,
		ActiveType: activeType,
		ActiveSeq:  activeSeq,
		ActiveSlug: activeSlug,
		Sidebar:    sidebarData(store, activeWS),
	}
}

// writePage renders a full page and writes it to the response.
func writePage(w http.ResponseWriter, store *db.Store, title, activeWS, activeType string, activeSeq int, activeSlug string, content string) {
	fmt.Fprint(w, render.Page(frame(store, title, activeWS, activeType, activeSeq, activeSlug), content))
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
		type result struct {
			Type      string `json:"type"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Summary   string `json:"summary"`
			Workspace string `json:"workspace"`
		}
		var results []result

		wsList, _ := store.GetWorkspaces()
		for _, w := range wsList {
			wsStore, err := store.Workspace(w.Name)
			if err != nil {
				continue
			}
			lessons, _ := wsStore.SearchLessons(q)
			for _, l := range lessons {
				results = append(results, result{
					Type: "lesson", Title: l.Title,
					URL:     fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber),
					Summary: l.Summary, Workspace: w.Name,
				})
			}
			recs, _ := wsStore.SearchRecords(q)
			for _, rec := range recs {
				results = append(results, result{
					Type: "record", Title: rec.Title,
					URL:     fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(w.Name), rec.SequenceNumber),
					Summary: rec.Summary, Workspace: w.Name,
				})
			}
			refs, _ := wsStore.SearchRefs(q)
			for _, ref := range refs {
				results = append(results, result{
					Type: "ref", Title: ref.Title,
					URL:     fmt.Sprintf("/workspace/%s/ref/%s", urlPathEscape(w.Name), urlPathEscape(ref.Slug)),
					Summary: ref.Summary, Workspace: w.Name,
				})
			}
		}
		if results == nil {
			results = []result{}
		}
		jsonResponse(w, results)
	}
}

// urlPathEscape replaces spaces for URL path segments.
func urlPathEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

// ── Dashboard ──

func handleAppShell(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
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
							continueURL = fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber)
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
						continueURL = fmt.Sprintf("/workspace/%s/ref/%s", urlPathEscape(w.Name), urlPathEscape(ref.Slug))
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

		writePage(w, store, "Dashboard", "", "", 0, "", render.Dashboard(data))
	}
}

// ── Workspace Page ──

func handleWorkspacePage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws := wsStore.Workspace()

		if ws.LastLessonSeq != nil {
			http.Redirect(w, r, fmt.Sprintf("/workspace/%s/lesson/%d", name, *ws.LastLessonSeq), http.StatusFound)
			return
		}

		lessons, _ := wsStore.GetLessons()
		records, _ := wsStore.GetRecords()
		refs, _ := wsStore.GetRefs()

		// Read mission from disk — the file is the source of truth,
		// not the DB column (which can go stale when the CLI writes
		// via --body-file or --edit without syncing back).
		// A mission with unresolved placeholders ({...}) counts as empty
		// — the workspace create command pre-populates the template.
		mission := ""
		if missionData, err := os.ReadFile(wsStore.Layout().MissionPath()); err == nil {
			if trimmed := strings.TrimSpace(string(missionData)); trimmed != "" && !strings.Contains(trimmed, "{") {
				mission = trimmed
			}
		}

		// Render mission markdown → HTML (same pattern as learning records)
		var missionHTML bytes.Buffer
		if mission != "" {
			if err := md.Convert([]byte(mission), &missionHTML); err != nil {
				missionHTML.WriteString("<p>Mission unavailable</p>")
			}
		}

		data := render.WorkspaceData{Workspace: ws, Mission: missionHTML.String(), Lessons: lessons, Records: records, Refs: refs}
		writePage(w, store, ws.DisplayName(), ws.Name, "", 0, "", render.Workspace(data))
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
	"glossary":  {title: "Glossary", path: db.Layout.GlossaryPath},
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
			http.NotFound(w, r)
			return
		}

		data := render.DocumentData{Title: dk.title, Kind: kind}

		raw, err := os.ReadFile(dk.path(wsStore.Layout()))
		if err == nil {
			trimmed := strings.TrimSpace(string(raw))
			// Workspace documents are seeded with placeholder templates on
			// create ({...} markers, or default prose for Notes). Treat an
			// unfilled template as empty so the learner gets guidance.
		hasPlaceholder := strings.Contains(trimmed, "{")
		_, isDefaultNotes := strings.CutPrefix(trimmed, "# Notes\n\nPreferences and working notes for this workspace.")
		isDefaultNotes = isDefaultNotes && kind == "notes"
			if trimmed != "" && !hasPlaceholder && !isDefaultNotes {
				// Strip a leading "# ..." H1 that duplicates the navbar title —
				// all document FORMAT templates start with one.
				if strings.HasPrefix(trimmed, "# ") {
					if nl := strings.IndexByte(trimmed, '\n'); nl >= 0 {
						trimmed = strings.TrimSpace(trimmed[nl+1:])
					} else {
						trimmed = ""
					}
				}
				if trimmed != "" {
					var buf bytes.Buffer
					if err := md.Convert([]byte(trimmed), &buf); err == nil {
						data.BodyHTML = buf.String()
					} else {
						data.BodyHTML = "<p>Document unavailable.</p>"
					}
				}
			}
		}
		wsStore.Touch()
		if data.BodyHTML == "" {
			data.Empty = true
		}

		writePage(w, store, dk.title, name, kind, 0, "", render.Document(data))
	}
}

// ── Lesson Page ──

func handleLessonPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		seq, _ := strconv.Atoi(r.PathValue("seq"))

		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		current, err := wsStore.GetLessonBySeq(seq)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Compute prev/next lesson in sequence order for in-content navigation.
		lessons, _ := wsStore.GetLessons()
		var prev, next *render.LessonNav
		for i, l := range lessons {
			if l.SequenceNumber == seq {
				if i > 0 {
					p := lessons[i-1]
					prev = &render.LessonNav{Seq: p.SequenceNumber, Title: p.Title, URL: fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(name), p.SequenceNumber)}
				}
				if i+1 < len(lessons) {
					n := lessons[i+1]
					next = &render.LessonNav{Seq: n.SequenceNumber, Title: n.Title, URL: fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(name), n.SequenceNumber)}
				}
				break
			}
		}

		data := render.LessonData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/lesson-html/%s/%s", urlPathEscape(name), urlPathEscape(current.Filename)),
			Seq:    seq,
			Total:  len(lessons),
			Prev:   prev,
			Next:   next,
		}
		wsStore.SetLastViewed("lesson", seq)
		writePage(w, store, current.Title, name, "lesson", seq, "", render.Lesson(data))
	}
}

// ── Record Page (MD → HTML) ──

func handleRecordPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		seq, _ := strconv.Atoi(r.PathValue("seq"))

		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws := wsStore.Workspace()

		current, err := wsStore.GetRecordBySeq(seq)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		recPath := filepath.Join(ws.Path, "learning-records", current.Filename)
		mdData, err := os.ReadFile(recPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		var buf bytes.Buffer
		if err := md.Convert(mdData, &buf); err != nil {
			buf.WriteString("<p>Error rendering markdown</p>")
		}

		data := render.RecordData{Title: current.Title, Status: current.Status, BodyHTML: buf.String()}
		wsStore.SetLastViewed("record", seq)
		writePage(w, store, current.Title, name, "record", seq, "", render.Record(data))
	}
}

// ── Reference View Page ──

func handleRefPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		slug := r.PathValue("slug")

		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		current, err := wsStore.GetRefBySlug(slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		data := render.RefData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/ref-html/%s/%s", urlPathEscape(name), urlPathEscape(current.Filename)),
		}
		wsStore.SetLastViewed("ref", 0)
		writePage(w, store, current.Title, name, "ref", 0, slug, render.Ref(data))
	}
}

// ── Search Page ──

func handleSearchPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		data := render.SearchData{Query: q}

		if q != "" {
			wsList, _ := store.GetWorkspaces()
			for _, w := range wsList {
				wsStore, err := store.Workspace(w.Name)
				if err != nil {
					continue
				}
				lessons, _ := wsStore.SearchLessons(q)
				for _, l := range lessons {
					data.Results = append(data.Results, render.SearchResult{
						Type: "lesson", Title: l.Title,
						URL:       fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber),
						Workspace: w.Name, Summary: l.Summary,
					})
				}
				recs, _ := wsStore.SearchRecords(q)
				for _, rec := range recs {
					data.Results = append(data.Results, render.SearchResult{
						Type: "record", Title: rec.Title,
						URL:       fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(w.Name), rec.SequenceNumber),
						Workspace: w.Name, Summary: rec.Summary,
					})
				}
				refs, _ := wsStore.SearchRefs(q)
				for _, ref := range refs {
					data.Results = append(data.Results, render.SearchResult{
						Type: "ref", Title: ref.Title,
						URL:       fmt.Sprintf("/workspace/%s/ref/%s", urlPathEscape(w.Name), urlPathEscape(ref.Slug)),
						Workspace: w.Name, Summary: ref.Summary,
					})
				}
			}
		}

		writePage(w, store, "Search", "", "", 0, "", render.Search(data))
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
	fmt.Fprint(w, fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Not found</title>
<style>
  :root {
    --bg: #eceff4;
    --card: #ffffff;
    --border: #e5e9f0;
    --muted: #6b7689;
    --text: #2e3440;
    --link: #5e81ac;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    font-family: 'Inter', ui-sans-serif, system-ui, -apple-system, sans-serif;
    background: var(--bg);
    color: var(--text);
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 2rem;
  }
  .card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 2.5rem 2rem;
    max-width: 26rem;
    text-align: center;
  }
  .icon {
    width: 48px;
    height: 48px;
    margin: 0 auto 1rem;
    color: var(--muted);
    opacity: 0.5;
  }
  h1 { font-size: 1.1rem; font-weight: 600; margin: 0 0 0.5rem; }
  p { font-size: 0.9rem; line-height: 1.6; color: var(--muted); margin: 0; }
  code {
    display: inline-block;
    margin-top: 1rem;
    background: var(--bg);
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-family: ui-monospace, 'SF Mono', monospace;
    color: var(--text);
  }
</style>
</head>
<body>
  <div class="card">
    <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>
      <path d="M10 17l5-5-5-5"/>
      <path d="M15 12H3"/>
    </svg>
    <h1>This %s isn&rsquo;t on disk</h1>
    <p>The workspace tracks this %s, but the file is missing. It may have been moved or deleted outside pharos. Re-revise it from your agent to restore the content.</p>
    <code>%s</code>
  </div>
</body>
</html>`, kind, kind, ident))
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
