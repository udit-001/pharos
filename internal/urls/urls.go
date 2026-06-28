package urls

import (
	"fmt"
	"net/url"
)

// PathEscape escapes a string for use as a URL path segment. It delegates
// to the stdlib url.PathEscape, which encodes the full reserved set (/ ? # %
// etc.) so a workspace name or slug containing those characters produces a
// valid, routeable URL. Go's ServeMux does not unescape %2F during route
// matching, so escaped slashes stay within a single {name} segment.
func PathEscape(s string) string {
	return url.PathEscape(s)
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

// QuizLibrary returns the quiz library page URL: /workspace/{name}/quizzes.
func QuizLibrary(ws string) string {
	return fmt.Sprintf("/workspace/%s/quizzes", PathEscape(ws))
}

// Quiz returns a quiz detail page URL: /workspace/{name}/quiz/{slug}.
func Quiz(ws, slug string) string {
	return fmt.Sprintf("/workspace/%s/quiz/%s", PathEscape(ws), PathEscape(slug))
}

// Doc returns a workspace document page URL (mission/resources/notes/glossary):
// /workspace/{name}/{kind}. kind is a fixed literal, so it is not escaped.
func Doc(ws, kind string) string {
	return fmt.Sprintf("/workspace/%s/%s", PathEscape(ws), kind)
}
