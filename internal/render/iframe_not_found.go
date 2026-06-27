package render

import (
	"bytes"
	"context"
)

// IframeNotFound renders the styled 404 page shown inside lesson/ref iframes
// when the tracked file is missing from disk. kind is the entity type
// ("lesson"/"ref"); ident is the missing filename or slug.
func IframeNotFound(kind, ident string) string {
	var buf bytes.Buffer
	if err := iframeNotFoundPage(kind, ident).Render(context.Background(), &buf); err != nil {
		return "<!DOCTYPE html><html><body><p>Not found: " + ident + "</p></body></html>"
	}
	return buf.String()
}
