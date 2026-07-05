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

// JS bundles served via /js/{file} for iframe injection.
//
//go:embed pharos-theme.js
var PharosThemeJS []byte
