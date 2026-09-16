package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udit-001/pharos/internal/config"
	"github.com/udit-001/pharos/internal/web"
)

func TestSetupPageVersionedBundleURL(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/setup")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /setup: %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}

	body := rec.Body.String()

	// Must contain the versioned pwa-install.bundle.js URL, not the
	// raw placeholder src that setup.html ships with.
	vURL := web.JSBundleURL("pwa-install.bundle.js")
	if vURL == "" {
		t.Fatal("JSBundleURL for pwa-install.bundle.js is empty")
	}
	if !strings.Contains(body, `src="`+vURL+`"`) {
		t.Errorf("body missing versioned src %q\nbody prefix: %.500s", vURL, body)
	}
	// The unversioned placeholder must have been replaced.
	if strings.Contains(body, `src="/js/pwa-install.bundle.js"`) {
		t.Errorf("body still contains unversioned placeholder src")
	}

	// Key content must be present (sanity: the card skeleton is there).
	for _, want := range []string{"Install Pharos", "pwa-install", "pwa-install.bundle.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestSetupPageNoCache(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/setup")
	if v := rec.Header().Get("Cache-Control"); !strings.Contains(v, "no-cache") {
		t.Errorf("Cache-Control = %q, want no-cache", v)
	}
}

func TestPWAInstalledPostWritesRecord(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	env := newTestEnv(t)

	rec := env.post(t, "/api/pwa/installed", `{"origin":"http://127.0.0.1:9090"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/pwa/installed: %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	var got config.PWAInstalled
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Installed {
		t.Error("installed = false, want true")
	}
	if got.Origin != "http://127.0.0.1:9090" {
		t.Errorf("origin = %q, want http://127.0.0.1:9090", got.Origin)
	}
	if got.Browser == "" {
		t.Error("browser field is empty")
	}
	if got.At == "" {
		t.Error("at field is empty")
	}

	// The file must be readable back via the config package.
	// ConfigDir is derived from XDG_CONFIG_HOME, matching the server's
	// process env (the CLI inherits the same env to the daemon).
	fetched, err := config.ReadPWAFile()
	if err != nil {
		t.Fatalf("ReadPWAFile after POST: %v", err)
	}
	if fetched == nil {
		t.Fatal("ReadPWAFile returned nil after POST")
	}
	if fetched.Origin != "http://127.0.0.1:9090" {
		t.Errorf("file origin = %q, want http://127.0.0.1:9090", fetched.Origin)
	}
}

func TestPWAInstalledPostBrowserSniffing(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want string
	}{
		{"chrome", "Mozilla/5.0 Chrome/128.0", "Chrome"},
		{"edge", "Mozilla/5.0 Edg/128.0", "Edge"},
		{"firefox", "Mozilla/5.0 Firefox/131.0", "Firefox"},
		{"safari", "Mozilla/5.0 Safari/605.1.15", "Safari"},
		{"crios", "Mozilla/5.0 CriOS/128.0 Mobile", "Chrome"},
		{"chromium", "Mozilla/5.0 Chromium/128.0", "Chrome"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			env := newTestEnv(t)
			rec := httptestRec(t, env.mux, "POST", "/api/pwa/installed", `{"origin":"http://127.0.0.1:9090"}`, tt.ua)
			if rec.Code != http.StatusOK {
				t.Fatalf("POST: %d, want 200", rec.Code)
			}
			var got config.PWAInstalled
			json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&got)
			if got.Browser != tt.want {
				t.Errorf("browser = %q for UA %q, want %q", got.Browser, tt.ua, tt.want)
			}
		})
	}
}

func TestPWAInstalledPostBadOrigin(t *testing.T) {
	env := newTestEnv(t)
	for _, body := range []string{
		`{}`,
		`{"origin":""}`,
		`{"origin":"ftp://not-http"}`,
		`{"origin":"nonsense"}`,
		`{"origin":"//no-scheme"}`,
	} {
		rec := env.post(t, "/api/pwa/installed", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST %s: %d, want 400", body, rec.Code)
		}
	}
}

func TestPWAInstalledGetAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	env := newTestEnv(t)
	rec := env.get(t, "/api/pwa/installed")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/pwa/installed: %d, want 404", rec.Code)
	}
}

func TestPWAInstalledGetReturnsRecord(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	env := newTestEnv(t)

	// Write a record via POST, then confirm GET returns it.
	post := env.post(t, "/api/pwa/installed", `{"origin":"http://127.0.0.1:9090"}`)
	if post.Code != http.StatusOK {
		t.Fatalf("POST: %d", post.Code)
	}

	get := env.get(t, "/api/pwa/installed")
	if get.Code != http.StatusOK {
		t.Fatalf("GET: %d, want 200", get.Code)
	}
	var got config.PWAInstalled
	if err := json.NewDecoder(strings.NewReader(get.Body.String())).Decode(&got); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if got.Origin != "http://127.0.0.1:9090" {
		t.Errorf("GET origin = %q, want http://127.0.0.1:9090", got.Origin)
	}
}

// httptestRec creates an httptest request with a User-Agent header.
func httptestRec(t *testing.T, mux *http.ServeMux, method, target, body, ua string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	mux.ServeHTTP(rec, req)
	return rec
}
