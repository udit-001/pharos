package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

// seedWorkbenchEvents creates test events via the store (simulating ingest).
func seedWorkbenchEvents(t *testing.T, store *db.Store, wsName string, events []db.WorkbenchEvent) {
	t.Helper()
	wsStore, err := store.Workspace(wsName)
	if err != nil {
		t.Fatalf("workspace %s: %v", wsName, err)
	}
	for _, ev := range events {
		if _, err := wsStore.AddWorkbenchEvent(ev.ID, ev.Namespace, ev.Type, ev.Ts, ev.Payload); err != nil {
			t.Fatalf("add event %s: %v", ev.ID, err)
		}
	}
}

func TestWorkbenchLog_Empty(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	out := runWithStore(t, []string{"workbench", "log", "-w", "ws1"}, store)
	if !strings.Contains(out, "no workbench events yet") {
		t.Errorf("expected empty message, got:\n%s", out)
	}
}

func TestWorkbenchLog_ShowsEvents(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	seedWorkbenchEvents(t, store, "ws1", []db.WorkbenchEvent{
		{ID: "ev-1", Namespace: "default", Type: "query", Ts: "2026-01-01T00:00:00Z", Payload: `{"sql":"SELECT 1","ok":true}`},
		{ID: "ev-2", Namespace: "default", Type: "error", Ts: "2026-01-02T00:00:00Z", Payload: `{"error":"syntax error"}`},
	})

	out := runWithStore(t, []string{"workbench", "log", "-w", "ws1"}, store)
	if !strings.Contains(out, "query") || !strings.Contains(out, "error") {
		t.Errorf("log missing events:\n%s", out)
	}
}

func TestWorkbenchLog_NeverTruncates(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	// Long SQL + long verbatim error — both must survive pretty mode intact
	// (LEARN-206 decision 6: one line per event, never truncated).
	longSQL := `SELECT region, customer_name, order_total, RANK() OVER (PARTITION BY region ORDER BY order_total DESC) AS regional_rank FROM monthly_sales_summary WHERE fiscal_quarter = '2026-Q3' ORDER BY regional_rank ASC, customer_name ASC LIMIT 25;`
	longErr := `SQLITE_ERROR: sqlite3 result code 1: no such column: monthly_sales_summary.regional_rank_after_discount_adjustment_with_tax_included`
	seedWorkbenchEvents(t, store, "ws1", []db.WorkbenchEvent{
		{ID: "ev-1", Namespace: "default", Type: "query", Ts: "2026-01-01T00:00:00Z", Payload: `{"sql":` + mustJSONString(t, longSQL) + `,"ok":true}`},
		{ID: "ev-2", Namespace: "default", Type: "query", Ts: "2026-01-02T00:00:00Z", Payload: `{"sql":"SELECT regon FROM books","ok":false,"error":` + mustJSONString(t, longErr) + `}`},
	})

	out := runWithStore(t, []string{"workbench", "log", "-w", "ws1"}, store)
	if !strings.Contains(out, longSQL) {
		t.Errorf("pretty log must carry the full SQL verbatim:\n%s", out)
	}
	if !strings.Contains(out, longErr) {
		t.Errorf("pretty log must carry the full error verbatim:\n%s", out)
	}
	if strings.Contains(out, "...") {
		t.Errorf("pretty log must not truncate:\n%s", out)
	}
}

// mustJSONString marshals s as a JSON string literal (with quotes) for
// building single-line payload fixtures in tests.
func mustJSONString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal %q: %v", s, err)
	}
	return string(b)
}

func TestWorkbenchLog_FilterType(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	seedWorkbenchEvents(t, store, "ws1", []db.WorkbenchEvent{
		{ID: "ev-1", Namespace: "default", Type: "query", Ts: "2026-01-01T00:00:00Z", Payload: `{"sql":"SELECT 1"}`},
		{ID: "ev-2", Namespace: "default", Type: "error", Ts: "2026-01-02T00:00:00Z", Payload: `{"error":"syntax error"}`},
	})

	out := runWithStore(t, []string{"workbench", "log", "-w", "ws1", "--type", "query"}, store)
	if strings.Contains(out, "error") {
		t.Errorf("log should not contain error event when filtering by query:\n%s", out)
	}
	if !strings.Contains(out, "query") {
		t.Errorf("log missing query event:\n%s", out)
	}
}

func TestWorkbenchLog_JSON(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	seedWorkbenchEvents(t, store, "ws1", []db.WorkbenchEvent{
		{ID: "ev-1", Namespace: "default", Type: "query", Ts: "2026-01-01T00:00:00Z", Payload: `{"sql":"SELECT 1","ok":true}`},
	})

	out := runWithStore(t, []string{"workbench", "log", "-w", "ws1", "--json"}, store)
	var m map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("log --json should output valid JSON, got:\n%s", out)
	}
	if m["id"] != "ev-1" {
		t.Errorf("json output id = %q, want ev-1", m["id"])
	}
	if m["type"] != "query" {
		t.Errorf("json output type = %q, want query", m["type"])
	}
}

func TestWorkbenchCheck_EventBatch(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Write a valid event batch file.
	eventsJSON := `{"events":[{"id":"ev-1","namespace":"default","type":"query","ts":"2026-01-01T00:00:00Z","payload":{"sql":"SELECT 1","ok":true}}]}`
	filePath := filepath.Join(t.TempDir(), "events.json")
	os.WriteFile(filePath, []byte(eventsJSON), 0644)

	// workbench check should succeed (exit 0) for valid events.
	out := runWithStore(t, []string{"workbench", "check", filePath}, store)
	if !strings.Contains(out, "✓") {
		t.Errorf("check should show success, got:\n%s", out)
	}
}

func TestWorkbenchCheck_InvalidEvent(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Write an invalid event batch (missing type).
	eventsJSON := `{"events":[{"id":"ev-1","ts":"2026-01-01T00:00:00Z","payload":{}}]}`
	filePath := filepath.Join(t.TempDir(), "bad-events.json")
	os.WriteFile(filePath, []byte(eventsJSON), 0644)

	// workbench check should exit 1 for invalid events.
	err := runWithStoreErr(t, []string{"workbench", "check", filePath}, store)
	if err == nil {
		t.Error("check should fail for invalid events")
	}
}

func TestWorkbenchCheck_Dataset(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	datasetJSON := `{"rows": [1, 2, 3]}`
	filePath := filepath.Join(t.TempDir(), "my-data.json")
	os.WriteFile(filePath, []byte(datasetJSON), 0644)

	out := runWithStore(t, []string{"workbench", "check", filePath}, store)
	if !strings.Contains(out, "✓") || !strings.Contains(out, "my-data") {
		t.Errorf("check dataset should succeed, got:\n%s", out)
	}
}

func TestWorkbenchAdd(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	dir := t.TempDir()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: dir})

	datasetJSON := `{"rows": [1, 2, 3]}`
	filePath := filepath.Join(t.TempDir(), "sample-data.json")
	os.WriteFile(filePath, []byte(datasetJSON), 0644)

	out := runWithStore(t, []string{"workbench", "add", filePath, "-w", "ws1"}, store)
	if !strings.Contains(out, "✓") || !strings.Contains(out, "sample-data") {
		t.Errorf("add should succeed, got:\n%s", out)
	}

	// File should now exist in workspace datasets/.
	dest := filepath.Join(dir, "datasets", "sample-data.json")
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Error("add should install dataset file into workspace")
	}
}

func TestWorkbenchRemove(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	dir := t.TempDir()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: dir})

	// Create a dataset file.
	datasetsDir := filepath.Join(dir, "datasets")
	os.MkdirAll(datasetsDir, 0755)
	filePath := filepath.Join(datasetsDir, "my-data.json")
	os.WriteFile(filePath, []byte(`{"key":"value"}`), 0644)

	out := runWithStore(t, []string{"workbench", "remove", "my-data", "-w", "ws1"}, store)
	if !strings.Contains(out, "✓") || !strings.Contains(out, "my-data") {
		t.Errorf("remove should succeed, got:\n%s", out)
	}

	// File should be gone.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("remove should delete the dataset file")
	}
}

func TestWorkbenchRemove_NotFound(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	store.AddWorkspace(db.Workspace{Name: "ws1", Path: t.TempDir()})

	err := runWithStoreErr(t, []string{"workbench", "remove", "nonexistent", "-w", "ws1"}, store)
	if err == nil {
		t.Error("remove should fail for nonexistent dataset")
	}
}

// runWithStoreErr is like runWithStore but returns the error instead of fatalf.
func runWithStoreErr(t *testing.T, args []string, store *db.Store) error {
	t.Helper()
	root := newRootForTest()
	ctx := context.WithValue(context.Background(), ctxStore{}, store)
	root.SetArgs(args)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	root.PersistentPostRunE = nil
	return root.ExecuteContext(ctx)
}
