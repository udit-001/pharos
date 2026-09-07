package web

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// ── Versioned asset URLs (universe A: embedded, immutable) ──────────────
//
// jsBundles is the single registry of JS bundles served at /js/{name}.
// JS (serving), JSBundleURL (script tags), and the version suffix all
// derive from this one map, so served bytes and referenced URLs can never
// drift apart. The version is the sha256 of the embedded bytes: a content
// change anywhere moves the URL everywhere, which makes the browser HTTP
// cache (max-age=86400) and the service worker's per-URL cache correct
// without manual version counters (the old jsVer=29 / ?v=28 / ?v=20
// literals that drifted across mux.go and frame.templ).
//
// Agent-created lesson assets (mermaid-theme.js, images, …) served under
// /api/lesson-html/*/assets/* are deliberately NOT here: they are mutable
// at a fixed URL (hrefs are written into lesson HTML by the agent), so
// versioning is the wrong tool. Those must keep flowing through the HTTP
// layer — http.ServeFile's Last-Modified / If-Modified-Since revalidation
// — and must not be cached first by the service worker.

var jsBundles = map[string][]byte{
	"pharos-theme.js":         PharosThemeJS,
	"pharos-toc.js":           PharosTocJS,
	"pharos-iframe-bridge.js": PharosIframeBridgeJS,
	"pharos-highlights.js":    PharosHighlightsJS,
	"pharos-scroll.js":        PharosScrollJS,
	"glossary-tooltip.js":     GlossaryTooltipJS,
	"presence.js":             PresenceJS,
}

var (
	versionOnce sync.Once
	jsVersions  map[string]string // bundle name -> "?v=<sha256[:12]>"
)

func assetVersion(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// JSBundleURL returns "/js/{name}?v=<sha256 of bytes>" for a registered
// bundle, or "" for an unknown name (callers drop unknown names, matching
// the old injectFrameScripts behavior). The URL is stable within a build
// and changes exactly when the bundle's content changes.
func JSBundleURL(name string) string {
	if _, ok := jsBundles[name]; !ok {
		return ""
	}
	versionOnce.Do(func() {
		jsVersions = make(map[string]string, len(jsBundles))
		for n, data := range jsBundles {
			jsVersions[n] = assetVersion(data)
		}
	})
	return "/js/" + name + "?v=" + jsVersions[name]
}

// JS returns the embedded bytes of a registered bundle for the /js/{file}
// server route, ok=false when the name is not registered.
func JS(name string) ([]byte, bool) {
	data, ok := jsBundles[name]
	return data, ok
}

var cssVersionOnce sync.Once
var cssVersion string

// CSSURL returns the cache-busting URL for /css/app.css, versioned by the
// embedded stylesheet's content.
func CSSURL() string {
	cssVersionOnce.Do(func() { cssVersion = assetVersion(CSS) })
	return "/css/app.css?v=" + cssVersion
}
