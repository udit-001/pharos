package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// md is the shared goldmark instance, configured with the external-link
// extender. Package-private — callers use Render.
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		&externalLinkExtender{},
	),
)

// Render converts markdown to HTML. On a render error (essentially
// unreachable for valid markdown input) it returns a generic fallback so
// callers never need to handle the error path individually.
func Render(text string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(text), &buf); err != nil {
		return "<p>Content unavailable.</p>"
	}
	return buf.String()
}
