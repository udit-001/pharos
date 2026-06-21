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
}

// ── Start ──

func Start(cfg Config) error {
	mux := http.NewServeMux()

	// Serve embedded Tailwind CSS
	mux.HandleFunc("GET /css/app.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
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
	mux.HandleFunc("GET /workspace/{name}/lesson/{seq}", handleLessonPage(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}/record/{seq}", handleRecordPage(cfg.DB))
	mux.HandleFunc("GET /workspace/{name}/ref/{seq}", handleRefPage(cfg.DB))
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
	return srv.Serve(listener)
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
func frame(store *db.Store, title, activeWS, activeType string, activeSeq int) render.Frame {
	return render.Frame{
		Title:      title,
		ActiveWS:   activeWS,
		ActiveType: activeType,
		ActiveSeq:  activeSeq,
		Sidebar:    sidebarData(store, activeWS),
	}
}

// writePage renders a full page and writes it to the response.
func writePage(w http.ResponseWriter, store *db.Store, title, activeWS, activeType string, activeSeq int, content string) {
	fmt.Fprint(w, render.Page(frame(store, title, activeWS, activeType, activeSeq), content))
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
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		ls, _ := store.GetLessons(id)
		if ls == nil {
			ls = []db.Lesson{}
		}
		jsonResponse(w, ls)
	}
}

func handleListRecords(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		recs, _ := store.GetLearningRecords(id)
		if recs == nil {
			recs = []db.LearningRecord{}
		}
		jsonResponse(w, recs)
	}
}

func handleListRefs(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		references, _ := store.GetReferences(id)
		if references == nil {
			references = []db.Reference{}
		}
		jsonResponse(w, references)
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

		ws, _ := store.GetWorkspaces()
		for _, w := range ws {
			ls, _ := store.SearchLessons(q, w.ID)
			for _, l := range ls {
				results = append(results, result{
					Type: "lesson", Title: l.Title,
					URL:       fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber),
					Summary:   l.Summary, Workspace: w.Name,
				})
			}
			recs, _ := store.SearchLearningRecords(q, w.ID)
			for _, rec := range recs {
				results = append(results, result{
					Type: "record", Title: rec.Title,
					URL:       fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(w.Name), rec.SequenceNumber),
					Summary:   rec.Summary, Workspace: w.Name,
				})
			}
			refs, _ := store.SearchReferences(q, w.ID)
			for _, ref := range refs {
				results = append(results, result{
					Type: "ref", Title: ref.Title,
					URL:       fmt.Sprintf("/workspace/%s/ref/%d", urlPathEscape(w.Name), ref.SequenceNumber),
					Summary:   ref.Summary, Workspace: w.Name,
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
				lessons, _ := store.GetLessons(w.ID)
				for _, l := range lessons {
					if l.SequenceNumber == *w.LastLessonSeq {
						continueURL = fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber)
						continueLabel = fmt.Sprintf("%s — Lesson: %s", w.Name, l.Title)
						break
					}
				}
			} else if w.LastRefSeq != nil && *w.LastRefSeq > 0 {
				references, _ := store.GetReferences(w.ID)
				for _, ref := range references {
					if ref.SequenceNumber == *w.LastRefSeq {
						continueURL = fmt.Sprintf("/workspace/%s/ref/%d", urlPathEscape(w.Name), ref.SequenceNumber)
						continueLabel = fmt.Sprintf("%s — Reference: %s", w.Name, ref.Title)
						break
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

		writePage(w, store, "Dashboard", "", "", 0, render.Dashboard(data))
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

		lessons, _ := wsStore.GetLessons()
		records, _ := wsStore.GetRecords()

		data := render.WorkspaceData{Workspace: ws, Mission: ws.MissionWhy, Lessons: lessons, Records: records}
		writePage(w, store, ws.DisplayName(), ws.Name, "", 0, render.Workspace(data))
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

		lessons, _ := wsStore.GetLessons()
		var current *db.Lesson
		for i := range lessons {
			if lessons[i].SequenceNumber == seq {
				current = &lessons[i]
				break
			}
		}
		if current == nil {
			http.NotFound(w, r)
			return
		}

		data := render.LessonData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/lesson-html/%s/%s", urlPathEscape(name), urlPathEscape(current.Filename)),
		}
		wsStore.SetLastViewed("lesson", seq)
		writePage(w, store, current.Title, name, "lesson", seq, render.Lesson(data))
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

		records, _ := wsStore.GetRecords()
		var current *db.LearningRecord
		for i := range records {
			if records[i].SequenceNumber == seq {
				current = &records[i]
				break
			}
		}
		if current == nil {
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
		writePage(w, store, current.Title, name, "record", seq, render.Record(data))
	}
}

// ── Reference View Page ──

func handleRefPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		seq, _ := strconv.Atoi(r.PathValue("seq"))

		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		references, _ := wsStore.GetRefs()
		var current *db.Reference
		for i := range references {
			if references[i].SequenceNumber == seq {
				current = &references[i]
				break
			}
		}
		if current == nil {
			http.NotFound(w, r)
			return
		}

		data := render.RefData{
			Title:  current.Title,
			RawURL: fmt.Sprintf("/api/ref-html/%s/%s", urlPathEscape(name), urlPathEscape(current.Filename)),
		}
		wsStore.SetLastViewed("ref", seq)
		writePage(w, store, current.Title, name, "ref", seq, render.Ref(data))
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
				ls, _ := store.SearchLessons(q, w.ID)
				for _, l := range ls {
					data.Results = append(data.Results, render.SearchResult{
						Type: "lesson", Title: l.Title,
						URL:       fmt.Sprintf("/workspace/%s/lesson/%d", urlPathEscape(w.Name), l.SequenceNumber),
						Workspace: w.Name, Summary: l.Summary,
					})
				}
				recs, _ := store.SearchLearningRecords(q, w.ID)
				for _, rec := range recs {
					data.Results = append(data.Results, render.SearchResult{
						Type: "record", Title: rec.Title,
						URL:       fmt.Sprintf("/workspace/%s/record/%d", urlPathEscape(w.Name), rec.SequenceNumber),
						Workspace: w.Name, Summary: rec.Summary,
					})
				}
				refs, _ := store.SearchReferences(q, w.ID)
				for _, ref := range refs {
					data.Results = append(data.Results, render.SearchResult{
						Type: "ref", Title: ref.Title,
						URL:       fmt.Sprintf("/workspace/%s/ref/%d", urlPathEscape(w.Name), ref.SequenceNumber),
						Workspace: w.Name, Summary: ref.Summary,
					})
				}
			}
		}

		writePage(w, store, "Search", "", "", 0, render.Search(data))
	}
}

// ── Raw lesson/reference file serving (for iframes) ──

func handleLessonHTML(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		file := r.PathValue("file")
		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws := wsStore.Workspace()
		http.ServeFile(w, r, filepath.Join(ws.Path, "lessons", file))
	}
}

func handleRefHTML(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		file := r.PathValue("file")
		wsStore, err := store.Workspace(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws := wsStore.Workspace()
		http.ServeFile(w, r, filepath.Join(ws.Path, "reference", file))
	}
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
