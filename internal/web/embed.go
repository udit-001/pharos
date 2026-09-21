package web

import _ "embed"

//go:embed app.css
var CSS []byte

//go:embed favicon.ico
var FaviconICO []byte

//go:embed favicon.png
var FaviconPNG []byte

//go:embed favicon.svg
var FaviconSVG []byte

//go:embed icon-192.png
var Icon192 []byte

//go:embed icon-512.png
var Icon512 []byte

//go:embed manifest.webmanifest
var Manifest []byte

//go:embed sw.js
var ServiceWorker []byte

//go:embed stopped.html
var StoppedPage []byte

// Vendored PWA install wizard component (UMD classic-script bundle, offline-safe).
//
//go:embed pwa-install.bundle.js
var PWAInstallBundleJS []byte

//go:embed setup.html
var SetupPage []byte

// JS bundles served via /js/{file} for iframe injection.
//
//go:embed pharos-theme.js
var PharosThemeJS []byte

//go:embed pharos-toc.js
var PharosTocJS []byte

//go:embed pharos-iframe-bridge.js
var PharosIframeBridgeJS []byte

//go:embed pharos-highlights.js
var PharosHighlightsJS []byte

//go:embed pharos-scroll.js
var PharosScrollJS []byte

// Copy-button logic for lesson code blocks marked `data-copy`. Injected on
// detection (see server/mux.go serveIframeHTML) — lessons never link it.
//
//go:embed copy-code.js
var CopyCodeJS []byte

//go:embed glossary-tooltip.js
var GlossaryTooltipJS []byte

// Behavior glue for features the server detects in lesson/reference/stimulus
// HTML (see server/detect.go). Idempotent — safe next to legacy lessons that
// still hand-wire their own copies.

//go:embed pharos-quiz.js
var PharosQuizJS []byte

//go:embed pharos-mermaid.js
var PharosMermaidJS []byte

//go:embed pharos-hljs.js
var PharosHljsJS []byte

//go:embed presence.js
var PresenceJS []byte
