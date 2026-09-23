package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/workbench"
)

// ─── Serve-time dataset resolution (LEARN-234) ──────────────────────

// datasetAttrRe matches the dataset attribute inside a single
// <sql-workbench ...> start tag (double-quoted value, tag-scoped: lessons
// escape their own code samples, so no false hits on prose).
var datasetAttrRe = regexp.MustCompile(`<sql-workbench\b[^>]*?\bdataset="([^"]*)"`)

// resolveDatasetSlugs rewrites a bare-slug dataset="..." attribute on a
// <sql-workbench> element to the workspace datasets route when the dataset
// is installed — the host composes route shapes so lesson content never
// does (LEARN-206 decision 8). Paths and URLs (the v0.5.1 contract) and
// uninstalled slugs pass through verbatim; the bench's plain-language boot
// error names the fix for the latter (assetScriptTag degrade precedent).
func resolveDatasetSlugs(html []byte, wsStore *db.WorkspaceStore) []byte {
	return datasetAttrRe.ReplaceAllFunc(html, func(m []byte) []byte {
		tag := datasetAttrRe.FindSubmatch(m)
		ref := string(tag[1])
		// Bare slug = anything the dataset-id pattern accepts; paths and
		// URLs contain / or . and fail it, so they pass through untouched.
		if workbench.ValidateDataset(ref) != nil {
			return m
		}
		ws := wsStore.Workspace()
		path := wsStore.Layout().DatasetPath(ref + ".json")
		if _, err := os.Stat(path); err != nil {
			log.Printf("[frame] %s: dataset %q not installed; leaving unresolved (bench boot error will name the fix)", ws.Name, ref)
			return m
		}
		resolved := "/api/workspaces/name/" + url.PathEscape(ws.Name) + "/datasets/" + ref
		return bytes.Replace(m, []byte(`dataset="`+ref+`"`), []byte(`dataset="`+resolved+`"`), 1)
	})
}

// ─── POST /api/workspaces/name/{name}/workbench-events ──────────────

// workbenchEventsRequest is the JSON body for the ingest endpoint.
type workbenchEventsRequest struct {
	Events []workbench.WorkbenchEvent `json:"events"`
}

// handleIngestWorkbenchEvents accepts a batch of bench events, validates
// them, and persists valid ones atomically (decision 2: all-or-nothing).
func handleIngestWorkbenchEvents(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			jsonError(w, "workspace not found", 404)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB safety cap
		if err != nil {
			jsonError(w, "failed to read request body", 400)
			return
		}

		var req workbenchEventsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			jsonError(w, "request body is not valid JSON", 400)
			return
		}

		if len(req.Events) == 0 {
			jsonError(w, "events array is empty", 400)
			return
		}

		// Batch cap: 50 events max (decision 2).
		const maxBatch = 50
		if len(req.Events) > maxBatch {
			jsonError(w, "batch exceeds maximum of 50 events", 400)
			return
		}

		// Validate all events first (all-or-nothing).
		for i, ev := range req.Events {
			if err := workbench.ValidateEvent(ev); err != nil {
				jsonError(w, "event["+itoa(i)+"]: "+err.Error(), 400)
				return
			}
		}

		// Persist all events.
		accepted := 0
		for _, ev := range req.Events {
			count, err := wsStore.AddWorkbenchEvent(
				ev.ID, ev.Namespace, ev.Type, ev.Ts, string(ev.Payload),
			)
			if err != nil {
				jsonError(w, "failed to store event: "+err.Error(), 500)
				return
			}
			accepted += int(count)
		}

		jsonResponse(w, map[string]int{"accepted": accepted})
	}
}

// ─── GET /api/workspaces/name/{name}/datasets/{id} ──────────────────

// handleGetDataset serves a dataset JSON file from the workspace's
// datasets/ directory. 50 MB cap; returns the raw bytes with
// application/json content type.
func handleGetDataset(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			jsonError(w, "workspace not found", 404)
			return
		}

		datasetID := r.PathValue("id")
		if err := workbench.ValidateDataset(datasetID); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		datasetsDir := wsStore.Layout().DatasetPath(".")
		filePath := wsStore.Layout().DatasetPath(datasetID + ".json")

		// Ensure the resolved path stays inside datasets/.
		rel, err := filepath.Rel(datasetsDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			jsonError(w, "invalid dataset path", 400)
			return
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				jsonError(w, "dataset not found", 404)
				return
			}
			jsonError(w, "failed to read dataset", 500)
			return
		}

		const maxBytes = 50 * 1024 * 1024
		if len(data) > maxBytes {
			jsonError(w, "dataset exceeds 50 MB limit", 413)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	}
}

// itoa converts a small non-negative int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
