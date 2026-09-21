package vendor

import (
	_ "embed"
)

// Embedded companions ship inside the binary next to their downloaded lib
// (theme glue, lightbox, render helpers). They are authored for exactly the
// pinned versions in manifest.go.

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

//go:embed vega-theme.js
var vegaThemeJS []byte

// Companions maps lib → embedded files served alongside its downloaded
// bytes. Served by the same /vendor route; never cached to disk.
var Companions = map[string]map[string][]byte{
	"mermaid": {
		"mermaid-theme.js":     mermaidThemeJS,
		"mermaid-lightbox.js":  mermaidLightboxJS,
		"mermaid-lightbox.css": mermaidLightboxCSS,
	},
	"katex": {
		"katex-render.js": katexRenderJS,
	},
	"vega": {
		"vega-theme.js": vegaThemeJS,
	},
	"highlightjs": {
		"highlight.css": highlightCSS,
	},
}
