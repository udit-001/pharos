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
