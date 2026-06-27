package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// externalLinkExtender adds target="_blank" rel="noopener noreferrer" to
// external (http/https) links so they open in a new tab — internal/relative
// links are left untouched.
type externalLinkExtender struct{}

func (e *externalLinkExtender) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&externalLinkRenderer{}, 0),
	))
}

type externalLinkRenderer struct{}

func (r *externalLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *externalLinkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString(`<a href="`)
		_, _ = w.Write(util.EscapeHTML(n.Destination))
		_ = w.WriteByte('"')
		if len(n.Title) > 0 {
			_, _ = w.WriteString(` title="`)
			_, _ = w.Write(util.EscapeHTML(n.Title))
			_ = w.WriteByte('"')
		}
		if isExternalURL(n.Destination) {
			_, _ = w.WriteString(` target="_blank" rel="noopener noreferrer"`)
		}
		_, _ = w.WriteString(">")
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}

func isExternalURL(dst []byte) bool {
	return bytes.HasPrefix(dst, []byte("http://")) || bytes.HasPrefix(dst, []byte("https://"))
}
