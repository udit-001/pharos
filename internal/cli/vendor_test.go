package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/vendor"
)

func testSHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func newTestCDN(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, string(data))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestVendorSyncCommand: `pharos vendor sync` fills the cache and reports
// progress, honoring the sync-options seam (temp dir + test CDN).
func TestVendorSyncCommand(t *testing.T) {
	dir := t.TempDir()
	content := []byte("/* mermaid lib */")
	oldManifest := vendor.Manifest
	vendor.Manifest = []vendor.Entry{{
		Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path:   "npm/mermaid@11.17.2/dist/mermaid.min.js",
		SHA256: testSHA256Hex(content),
	}}
	t.Cleanup(func() { vendor.Manifest = oldManifest })

	srv := newTestCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": content})
	oldOpts := vendorSyncOpts
	vendorSyncOpts = func() vendor.SyncOptions {
		return vendor.SyncOptions{Dir: dir, BaseURL: srv.URL, Out: os.Stdout}
	}
	t.Cleanup(func() { vendorSyncOpts = oldOpts })

	out := runWithStore(t, []string{"vendor", "sync"}, nil)
	if !strings.Contains(out, "[vendor] fetching mermaid@11.17.2") {
		t.Errorf("expected progress line in output, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "mermaid@11.17.2", "mermaid.min.js")); err != nil {
		t.Errorf("vendor sync should fill the cache: %v", err)
	}
}

// TestStartSyncsVendorCache: `pharos start` fills the vendor cache
// synchronously before the server boots (the only code path every install
// channel is guaranteed to execute). Boots the real foreground server on a
// hermetic port, polls for the cached file, then SIGTERMs (the server's
// signal handler shuts it down cleanly and Execute returns).
func TestStartSyncsVendorCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	port := 19741

	content := []byte("/* mermaid lib */")
	oldManifest := vendor.Manifest
	vendor.Manifest = []vendor.Entry{{
		Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
		Path:   "npm/mermaid@11.17.2/dist/mermaid.min.js",
		SHA256: testSHA256Hex(content),
	}}
	t.Cleanup(func() { vendor.Manifest = oldManifest })

	srv := newTestCDN(t, map[string][]byte{"npm/mermaid@11.17.2/dist/mermaid.min.js": content})
	oldOpts := vendorSyncOpts
	vendorSyncOpts = func() vendor.SyncOptions {
		return vendor.SyncOptions{BaseURL: srv.URL, Out: os.Stdout}
	}
	t.Cleanup(func() { vendorSyncOpts = oldOpts })

	store, cleanup := newTestStore(t)
	defer cleanup()

	// Boot start --foreground in-process; its sync runs before the listener.
	done := make(chan error, 1)
	go func() {
		root := newRootForTest()
		root.SetArgs([]string{"start", "--foreground", "--no-open", "--port", strconv.Itoa(port)})
		ctx := context.WithValue(context.Background(), ctxStore{}, store)
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
			return nil
		}
		done <- root.ExecuteContext(ctx)
	}()

	cached := filepath.Join(vendor.DefaultDir(), "mermaid@11.17.2", "mermaid.min.js")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(cached); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("start did not fill the vendor cache: %v", err)
	}

	// Shut the server down: its signal handler consumes SIGTERM and returns.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("sigterm: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("start did not shut down after SIGTERM")
	}
}

// TestStartAlreadyRunningSkipsSync: when a server is already up, start
// returns immediately — no sync pass, no fetches (cheap idempotent path).
func TestStartAlreadyRunningSkipsSync(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	runningPort, configPort := 19742, 19742
	writeConfigWithPort(t, configPort)
	writePidFile(t, runningPort, os.Getpid())
	defer fakeServer(t, runningPort)()

	oldOpts := vendorSyncOpts
	hits := 0
	vendorSyncOpts = func() vendor.SyncOptions {
		hits++
		return vendor.SyncOptions{Out: os.Stdout}
	}
	t.Cleanup(func() { vendorSyncOpts = oldOpts })

	out := captureCLI(t, []string{"start"})
	if !strings.Contains(out, "already running") {
		t.Fatalf("start output = %q", out)
	}
	if hits != 0 {
		t.Error("already-running start must not run a vendor sync pass")
	}
}
