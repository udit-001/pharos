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

// fetchLib downloads the vendored lib for spec over HTTP. Returns nil for
// embedded-only assets (no URLTemplate). When force is false (idempotent add)
// it skips the fetch if the lib is already present — the store skips writing
// it too. When force is true (redeploy) it always fetches so the lib is
// overwritten to the current pinned version.
//
// The HTTP fetch (a true-external dependency) stays in the CLI tier; the
// store never touches the network. The store owns the per-file skip/overwrite
// policy; the CLI owns the fetch decision.
func fetchLib(spec db.AssetSpec, wsStore *db.WorkspaceStore, force bool) ([]byte, error) {
	if spec.URLTemplate == "" {
		return nil, nil
	}
	if !force && libPresent(spec, wsStore) {
		return nil, nil
	}
	url := strings.ReplaceAll(spec.URLTemplate, "{{VERSION}}", spec.DefaultVersion)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", spec.Filename, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: server returned %s", spec.Filename, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return data, nil
}

// installJSON is the machine-readable shape for add/redeploy results.
type installJSON struct {
	Name     string   `json:"name"`
	Version  string   `json:"version,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Written  []string `json:"written"`
	Skipped  []string `json:"skipped"`
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
