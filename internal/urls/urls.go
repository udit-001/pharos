package urls

import (
	"fmt"
	"strings"
)

// PathEscape replaces spaces for URL path segments. This preserves the
// historical behaviour of the three former urlPathEscape helpers exactly —
// it is intentionally space-only. Path-escape correctness for the full
// reserved set is tracked separately (see LEARN-51).
func PathEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

// Workspace returns the workspace page URL: /workspace/{name}.
func Workspace(name string) string {
	return fmt.Sprintf("/workspace/%s", PathEscape(name))
}

// Lesson returns a lesson page URL: /workspace/{name}/lesson/{seq}.
func Lesson(ws string, seq int) string {
	return fmt.Sprintf("/workspace/%s/lesson/%d", PathEscape(ws), seq)
}

// Record returns a record page URL: /workspace/{name}/record/{seq}.
func Record(ws string, seq int) string {
	return fmt.Sprintf("/workspace/%s/record/%d", PathEscape(ws), seq)
}

// Ref returns a reference page URL: /workspace/{name}/ref/{slug}.
func Ref(ws, slug string) string {
	return fmt.Sprintf("/workspace/%s/ref/%s", PathEscape(ws), PathEscape(slug))
}

// Doc returns a workspace document page URL (mission/resources/notes/glossary):
// /workspace/{name}/{kind}. kind is a fixed literal, so it is not escaped.
func Doc(ws, kind string) string {
	return fmt.Sprintf("/workspace/%s/%s", PathEscape(ws), kind)
}
