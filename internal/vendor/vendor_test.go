package vendor

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testCDN serves bytes for the manifest's test entry. Tests build their own
// tiny manifest via withTestManifest so this stays self-contained.
func testCDN(t *testing.T, files map[string][]byte, hits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			*hits++
		}
		data, ok := files[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// withTestManifest swaps the real manifest for a one-entry test manifest and
// restores it when the test ends.
func withTestManifest(t *testing.T, entries []Entry) {
	t.Helper()
	old := Manifest
	Manifest = entries
	t.Cleanup(func() { Manifest = old })
}

// TestSyncFetchesAndVerifies: a fresh cache dir + reachable CDN → the lib's
// file lands in the cache at <lib>@<version>/<file> and matches the pin.
func TestSyncFetchesAndVerifies(t *testing.T) {
	dir := t.TempDir()
	content := []byte("/* mermaid lib */")
	entry := Entry{
		Lib:     "mermaid",
		Version: "11.17.2",
		File:    "mermaid.min.js",
		Path:    "npm/mermaid@11.17.2/dist/mermaid.min.js",
		SHA256:  sha256hex(content),
	}
	withTestManifest(t, []Entry{entry})
	var hits int
	srv := testCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": content}, &hits)

	res, err := Sync(SyncOptions{Dir: dir, BaseURL: srv.URL, Out: &strings.Builder{}})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected exactly 1 fetch, got %d", hits)
	}
	if res.Fetched != 1 {
		t.Errorf("expected 1 fetched file, got %d", res.Fetched)
	}
	got, err := os.ReadFile(filepath.Join(dir, "mermaid@11.17.2", "mermaid.min.js"))
	if err != nil {
		t.Fatalf("cached file missing: %v", err)
	}
	if string(got) != string(content) {
		t.Error("cached bytes differ from CDN bytes")
	}
}

// TestSyncIsIdempotent: a cache that already matches the pin is verified,
// not refetched — zero HTTP requests on the second pass.
func TestSyncIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	content := []byte("/* lib */")
	entry := Entry{Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path: "npm/mermaid@11.17.2/dist/mermaid.min.js", SHA256: sha256hex(content)}
	withTestManifest(t, []Entry{entry})
	var hits int
	srv := testCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": content}, &hits)

	opts := SyncOptions{Dir: dir, BaseURL: srv.URL, Out: &strings.Builder{}}
	if _, err := Sync(opts); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	res, err := Sync(opts)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if hits != 1 {
		t.Errorf("second sync must not fetch; total requests = %d", hits)
	}
	if res.Verified != 1 || res.Fetched != 0 {
		t.Errorf("second sync should verify, not fetch: %+v", res)
	}
}

// TestSyncRefetchesTamperedCache: a cache file whose hash no longer matches
// the pin is refetched and restored to the pinned bytes.
func TestSyncRefetchesTamperedCache(t *testing.T) {
	dir := t.TempDir()
	content := []byte("/* pinned bytes */")
	entry := Entry{Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path: "npm/mermaid@11.17.2/dist/mermaid.min.js", SHA256: sha256hex(content)}
	withTestManifest(t, []Entry{entry})
	var hits int
	srv := testCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": content}, &hits)

	if err := os.MkdirAll(filepath.Join(dir, "mermaid@11.17.2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mermaid@11.17.2", "mermaid.min.js"), []byte("/* tampered */"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Sync(SyncOptions{Dir: dir, BaseURL: srv.URL, Out: &strings.Builder{}})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if hits != 1 {
		t.Errorf("tampered cache should trigger exactly one refetch; requests = %d", hits)
	}
	got, err := os.ReadFile(filepath.Join(dir, "mermaid@11.17.2", "mermaid.min.js"))
	if err != nil || string(got) != string(content) {
		t.Error("tampered cache file was not restored to pinned bytes")
	}
	if res.Fetched != 1 || res.Verified != 0 {
		t.Errorf("expected 1 fetched, 0 verified; got %+v", res)
	}
}

// TestSyncPinMismatchSkips: the CDN serves bytes that don't match the pin —
// the file is skipped and logged, Sync returns no error, and the cache keeps
// whatever it had (nothing, here).
func TestSyncPinMismatchSkips(t *testing.T) {
	dir := t.TempDir()
	entry := Entry{Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path: "npm/mermaid@11.17.2/dist/mermaid.min.js", SHA256: sha256hex([]byte("/* pinned */"))}
	withTestManifest(t, []Entry{entry})
	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	srv := testCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": []byte("/* drifted */")}, nil)
	res, err := Sync(SyncOptions{Dir: dir, BaseURL: srv.URL, Out: &strings.Builder{}})
	if err != nil {
		t.Fatalf("pin mismatch must degrade, not fail: %v", err)
	}
	if res.Skipped != 1 || res.Fetched != 0 {
		t.Errorf("expected 1 skipped, 0 fetched; got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "mermaid@11.17.2", "mermaid.min.js")); !os.IsNotExist(err) {
		t.Error("mismatched bytes must not enter the cache")
	}
	if !strings.Contains(logBuf.String(), "sha256 mismatch") {
		t.Error("pin mismatch should be logged")
	}
}

// TestSyncOfflineDegrades: an unreachable CDN (fresh cache) → every file
// skipped + logged, Sync returns no error — start must succeed offline.
func TestSyncOfflineDegrades(t *testing.T) {
	dir := t.TempDir()
	entry := Entry{Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path: "npm/mermaid@11.17.2/dist/mermaid.min.js", SHA256: sha256hex([]byte("x"))}
	withTestManifest(t, []Entry{entry})
	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	// Port 1 on localhost is a reliable connection-refused.
	res, err := Sync(SyncOptions{Dir: dir, BaseURL: "http://127.0.0.1:1", Out: &strings.Builder{}})
	if err != nil {
		t.Fatalf("offline sync must degrade, not fail: %v", err)
	}
	if res.Skipped != 1 || res.Fetched != 0 {
		t.Errorf("expected 1 skipped, 0 fetched; got %+v", res)
	}
	if !strings.Contains(logBuf.String(), "[vendor]") {
		t.Error("offline skip should be logged with the [vendor] prefix")
	}
}

// TestSyncBudgetExhausted: a CDN slower than the budget leaves later files
// skipped (logged) and Sync still returns without error.
func TestSyncBudgetExhausted(t *testing.T) {
	dir := t.TempDir()
	c := []byte("/* lib */")
	e := func(file string) Entry {
		return Entry{Lib: "big", Version: "1.0.0", File: file,
			Path: "npm/big@1.0.0/dist/" + file, SHA256: sha256hex(c)}
	}
	withTestManifest(t, []Entry{e("one.js"), e("two.js")})
	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write(c)
	}))
	t.Cleanup(srv.Close)

	res, err := Sync(SyncOptions{Dir: dir, BaseURL: srv.URL, Budget: 50 * time.Millisecond, Out: &strings.Builder{}})
	if err != nil {
		t.Fatalf("budget exhaustion must degrade, not fail: %v", err)
	}
	if res.Fetched+res.Skipped != 2 {
		t.Errorf("every file must be fetched or skipped: %+v", res)
	}
	if res.Fetched > 1 {
		t.Errorf("budget should have capped fetching at ~1 file: %+v", res)
	}
	if !strings.Contains(logBuf.String(), "budget exhausted") {
		t.Error("budget skips should be logged")
	}
}
