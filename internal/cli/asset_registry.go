package cli

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/udit-001/pharos/internal/db"
)

// Vendored companion embeds — embedded files that ship alongside a
// downloaded lib (mermaid-theme.js, highlight.css, the lightbox files). The
// seeded universal files (style.css, glossary-tooltip.js, copy-code.js, the
// Inter font) are embedded in internal/db/seed.go and referenced below as
// db.SeedStyleCSS etc.

//go:embed highlight.css
var highlightCSS []byte

//go:embed mermaid-lightbox.css
var mermaidLightboxCSS []byte

//go:embed mermaid-lightbox.js
var mermaidLightboxJS []byte

//go:embed mermaid-theme.js
var mermaidThemeJS []byte

//go:embed katex-render.js
var katexRenderJS []byte

// knownAssets is the registry of installable assets — vendored (a downloaded
// lib plus embedded companions) and seeded (embedded-only universal files).
// The CLI builds db.AssetSpec values from this map; the store installs them
// without knowing the source. Source is informational (seeded vs vendored);
// add/redeploy handle both uniformly.
var knownAssets = map[string]db.AssetSpec{
	"mermaid": {
		Source:         "vendored",
		Filename:       "mermaid.min.js",
		DefaultVersion: "11",
		URLTemplate:    "https://cdn.jsdelivr.net/npm/mermaid@{{VERSION}}/dist/mermaid.min.js",
		Files:          map[string][]byte{"mermaid-theme.js": mermaidThemeJS},
	},
	"mermaid-lightbox": {
		Source: "vendored",
		Files: map[string][]byte{
			"mermaid-lightbox.js":  mermaidLightboxJS,
			"mermaid-lightbox.css": mermaidLightboxCSS,
		},
	},
	"highlightjs": {
		Source:         "vendored",
		Filename:       "highlight.min.js",
		DefaultVersion: "11.11.1",
		URLTemplate:    "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@{{VERSION}}/build/highlight.min.js",
		Files:          map[string][]byte{"highlight.css": highlightCSS},
	},
	"katex": {
		Source:         "vendored",
		Filename:       "katex.min.js",
		DefaultVersion: "0.16.22",
		URLTemplate:    "https://cdn.jsdelivr.net/npm/katex@{{VERSION}}/dist/katex.min.js",
		Files:          map[string][]byte{"katex-render.js": katexRenderJS},
		Downloads:      katexDownloads,
	},
	"style": {
		Source: "seeded",
		Files:  map[string][]byte{"style.css": []byte(db.SeedStyleCSS)},
	},
	"glossary-tooltip": {
		Source: "seeded",
		Files:  map[string][]byte{"glossary-tooltip.js": []byte(db.SeedGlossaryTooltipJS)},
	},
	"copy-code": {
		Source: "seeded",
		Files:  map[string][]byte{"copy-code.js": []byte(db.SeedCopyCodeJS)},
	},
	"inter-font": {
		Source: "seeded",
		Files:  map[string][]byte{"fonts/inter-latin.woff2": db.SeedInterLatinWOFF2},
	},
}

// katexWoff2Fonts are the 20 font families KaTeX 0.16.x ships as woff2.
// KaTeX's CSS references woff2 first (format hint), so woff/ttf are not
// needed — the browser stops at the first src it can load.
var katexWoff2Fonts = []string{
	"KaTeX_AMS-Regular",
	"KaTeX_Caligraphic-Bold",
	"KaTeX_Caligraphic-Regular",
	"KaTeX_Fraktur-Bold",
	"KaTeX_Fraktur-Regular",
	"KaTeX_Main-Bold",
	"KaTeX_Main-BoldItalic",
	"KaTeX_Main-Italic",
	"KaTeX_Main-Regular",
	"KaTeX_Math-BoldItalic",
	"KaTeX_Math-Italic",
	"KaTeX_SansSerif-Bold",
	"KaTeX_SansSerif-Italic",
	"KaTeX_SansSerif-Regular",
	"KaTeX_Script-Regular",
	"KaTeX_Size1-Regular",
	"KaTeX_Size2-Regular",
	"KaTeX_Size3-Regular",
	"KaTeX_Size4-Regular",
	"KaTeX_Typewriter-Regular",
}

// katexDownloads is the URL template map for KaTeX's extra network files
// (CSS, auto-render, fonts). Built from katexWoff2Fonts so the registry
// literal below is self-contained. KaTeX's CSS uses relative font URLs
// (url(fonts/...)), which resolve against the CSS location — fully local.
var katexDownloads = func() map[string]string {
	const base = "https://cdn.jsdelivr.net/npm/katex@{{VERSION}}/dist"
	dl := map[string]string{
		"katex.min.css":              base + "/katex.min.css",
		"contrib/auto-render.min.js": base + "/contrib/auto-render.min.js",
	}
	for _, f := range katexWoff2Fonts {
		dl["fonts/"+f+".woff2"] = base + "/fonts/" + f + ".woff2"
	}
	return dl
}()

// knownAssetsString returns a sorted, comma-separated list of registry names
// for error messages.
func knownAssetsString() string {
	names := make([]string, 0, len(knownAssets))
	for k := range knownAssets {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// libPresent reports whether the vendored lib file for spec is already in the
// workspace's assets/ directory. Returns false for embedded-only assets.
func libPresent(spec db.AssetSpec, wsStore *db.WorkspaceStore) bool {
	if spec.Filename == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(wsStore.Workspace().Path, "assets", spec.Filename))
	return err == nil
}

// fetchURL downloads a single file over HTTP, substituting {{VERSION}} in
// urlTmpl with version. label names the file in error messages. The HTTP
// fetch (a true-external dependency) stays in the CLI tier; the store never
// touches the network.
func fetchURL(urlTmpl, label, version string) ([]byte, error) {
	url := strings.ReplaceAll(urlTmpl, "{{VERSION}}", version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: server returned %s", label, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return data, nil
}

// fetchLib downloads the vendored lib for spec. Returns nil for embedded-only
// assets (no URLTemplate). When force is false (idempotent add) it skips the
// fetch if the lib is already present. When force is true (redeploy) it always
// fetches so the lib is overwritten to the current pinned version.
func fetchLib(spec db.AssetSpec, wsStore *db.WorkspaceStore, force bool) ([]byte, error) {
	if spec.URLTemplate == "" {
		return nil, nil
	}
	if !force && libPresent(spec, wsStore) {
		return nil, nil
	}
	return fetchURL(spec.URLTemplate, spec.Filename, spec.DefaultVersion)
}

// fetchDownloads fetches each extra download file (spec.Downloads) over HTTP.
// Returns nil for assets with no Downloads. When force is false, each file is
// fetched only if absent from the workspace — per-file idempotency, mirroring
// the store's own skip logic.
func fetchDownloads(spec db.AssetSpec, wsStore *db.WorkspaceStore, force bool) (map[string][]byte, error) {
	if len(spec.Downloads) == 0 {
		return nil, nil
	}
	assetsDir := filepath.Join(wsStore.Workspace().Path, "assets")
	result := make(map[string][]byte)
	for filename, urlTmpl := range spec.Downloads {
		if !force {
			if _, err := os.Stat(filepath.Join(assetsDir, filename)); err == nil {
				continue
			}
		}
		data, err := fetchURL(urlTmpl, filename, spec.DefaultVersion)
		if err != nil {
			return nil, err
		}
		result[filename] = data
	}
	return result, nil
}

// resolve fetches all network files for a vendored asset — the primary lib
// plus any Downloads — and returns a spec whose Files map is complete
// (embedded companions merged with fetched downloads), ready for InstallAsset.
// libData is returned separately so the store can report LibWritten.
// force=false skips per-file fetches for files already present; force=true
// re-fetches everything. The store owns the per-file skip/overwrite policy;
// the CLI owns the fetch decision.
func resolve(spec db.AssetSpec, wsStore *db.WorkspaceStore, force bool) (db.AssetSpec, []byte, error) {
	libData, err := fetchLib(spec, wsStore, force)
	if err != nil {
		return spec, nil, err
	}
	downloads, err := fetchDownloads(spec, wsStore, force)
	if err != nil {
		return spec, nil, err
	}
	if len(downloads) > 0 {
		spec.Files = mergeDownloads(spec.Files, downloads)
	}
	return spec, libData, nil
}

// installJSON is the machine-readable shape for add/redeploy results.
type installJSON struct {
	Name     string   `json:"name"`
	Version  string   `json:"version,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Written  []string `json:"written"`
	Skipped  []string `json:"skipped"`
}

// mergeDownloads returns a new map combining embedded companions (files) with
// fetched downloads. A fresh map avoids mutating the shared knownAssets entry.
func mergeDownloads(files map[string][]byte, downloads map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(files)+len(downloads))
	for k, v := range files {
		out[k] = v
	}
	for k, v := range downloads {
		out[k] = v
	}
	return out
}

// printInstall renders an InstallAsset result for add/redeploy. verb is
// "Added" or "Redeployed".
func printInstall(name string, spec db.AssetSpec, res db.AssetResult, verb string, ws db.Workspace) {
	written := res.FilesWritten
	if res.LibWritten && spec.Filename != "" {
		written = append([]string{spec.Filename}, written...)
	}
	if jsonOut {
		printJSON(installJSON{
			Name:     name,
			Version:  spec.DefaultVersion,
			Filename: spec.Filename,
			Written:  written,
			Skipped:  res.Skipped,
		})
		return
	}
	fmt.Println()
	if len(written) == 0 {
		fmt.Printf("  ✓ %s already present — skipping\n", name)
		fmt.Println()
		return
	}
	fmt.Printf("  ✓ %s %s\n", verb, name)
	for _, f := range written {
		fmt.Printf("    File: %s\n", filepath.Join(ws.Path, "assets", f))
	}
	fmt.Println()
}
