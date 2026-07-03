package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNotesRenderRepro drives the exact user symptom: notes written to disk
// must appear in the web UI. Cases cover realistic note contents. A failing
// case here reproduces "notes don't appear in web UI".
func TestNotesRenderRepro(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string // substring that MUST appear in rendered HTML
	}{
		{"plain-prose", "# Notes\n\nI prefer dark mode and concise answers.", "prefer dark mode"},
		{"json-config", "# Notes\n\nMy config:\n\n{\"theme\": \"dark\"}", "My config"},
		{"code-block", "# Notes\n\nExample:\n\n```go\nfunc main() {}\n```", "func main"},
		{"inline-code-brace", "# Notes\n\nUse map[string]int{} for counts.", "for counts"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newTestEnv(t)
			if err := os.WriteFile(filepath.Join(env.wsDir, "NOTES.md"), []byte(c.content), 0644); err != nil {
				t.Fatal(err)
			}
			rec := env.get(t, "/workspace/alpha/notes")
			if rec.Code != 200 {
				t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
			}
			body := rec.Body.String()
			if !strings.Contains(body, c.want) {
				t.Errorf("notes page did not render note content %q.\nwant substring %q absent.\n--- empty-state present? %v ---",
					c.content, c.want, strings.Contains(body, "icon-book-open"))
			}
		})
	}
}
