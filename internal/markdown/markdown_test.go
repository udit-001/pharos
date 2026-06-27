package markdown

import "testing"

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "paragraph", in: "Hello world", want: "<p>Hello world</p>\n"},
		{name: "heading", in: "# Title", want: "<h1>Title</h1>\n"},
		{name: "empty", in: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Render(tc.in); got != tc.want {
				t.Errorf("Render(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExternalLinkExtender(t *testing.T) {
	got := Render("[click](https://example.com)")
	if !contains(got, `target="_blank"`) || !contains(got, `rel="noopener noreferrer"`) {
		t.Errorf("external link missing target/rel attrs: %s", got)
	}
}

func TestInternalLinkUntouched(t *testing.T) {
	got := Render("[home](/workspace/foo)")
	if contains(got, `target="_blank"`) {
		t.Errorf("internal link got target=_blank: %s", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
