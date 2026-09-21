package server

import (
	"bytes"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// Frame feature detection: what a lesson/reference/stimulus document actually
// uses, decided by HTML-token matching (CSS selectors via goquery) — never by
// substring checks. The old bytes.Contains approach false-positived on prose
// and code samples mentioning feature names ("mark snippets with data-copy").
//
// Unexported on purpose: the test surface is the HTTP seam (which tags land in
// the served HTML), not this struct.

// frameFeatures reports which injectable features a document uses.
type frameFeatures struct {
	StyleLinked bool // already links assets/style.css (legacy page)
	Glossary    bool // .glossary-term elements
	CopyCode    bool // pre[data-copy]
	Quiz        bool // .q elements
	Mermaid     bool // .mermaid containers
	Vega        bool // [data-vega] charts
	Workbench   bool // <sql-workbench> custom element
	Highlight   bool // pre code with a language-* class (opt-in)
	Katex       bool // math delimiters in prose text
}

// katexRe matches math delimiters in prose: \(...\), \[...\], $$...$$, and
// $...$ guarded against digits (so "costs $5 and $10" never fires). Text
// inside pre/code/script/style is excluded before matching — the same tags
// KaTeX's auto-renderer ignores.
var katexRe = regexp.MustCompile(`\\\(|\\\[|\$\$|\$[^\s$0-9\\][^$\n]*?\$`)

// detectFrameFeatures parses the document and probes each feature with a
// token-accurate selector.
func detectFrameFeatures(html []byte) frameFeatures {
	var f frameFeatures
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return f
	}

	f.StyleLinked = doc.Find(`link[rel="stylesheet"][href*="style.css"]`).Length() > 0
	f.Glossary = doc.Find(".glossary-term").Length() > 0
	f.CopyCode = doc.Find("pre[data-copy]").Length() > 0
	f.Quiz = doc.Find(".q").Length() > 0
	f.Mermaid = doc.Find(".mermaid").Length() > 0
	f.Vega = doc.Find("[data-vega]").Length() > 0
	f.Workbench = doc.Find("sql-workbench").Length() > 0
	f.Highlight = doc.Find(`pre code[class*="language-"]`).Length() > 0

	// KaTeX: scan prose text only (exclude the tags auto-render ignores, and
	// noscript per the repo's text-extraction lesson — its content is inert
	// but still text).
	prose := doc.Find("body").Clone()
	prose.Find("pre, code, script, style, noscript").Remove()
	f.Katex = katexRe.MatchString(prose.Text())

	return f
}
