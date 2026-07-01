package docutil

import "testing"

func TestIsTemplate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		kind    string
		want    bool
	}{
		{name: "empty", content: "", kind: "mission", want: true},
		{name: "placeholder mission", content: "# Mission: {Display}\n\n## Why\n{fill in}", kind: "mission", want: true},
		{name: "filled mission", content: "# Mission: SQL\n\n## Why\nTo query.", kind: "mission", want: false},
		{name: "filled resources", content: "# SQL Resources\n\n- [link](url)", kind: "resources", want: false},
		{name: "default notes", content: "# Notes\n\nPreferences and working notes for this workspace.", kind: "notes", want: true},
		{name: "edited notes", content: "# Notes\n\nI prefer dark mode.", kind: "notes", want: false},
		{name: "appended notes", content: "# Notes\n\nPreferences and working notes for this workspace.\n- Agent note: learned about go channels", kind: "notes", want: false},
		{name: "default notes prose but kind mission", content: "# Notes\n\nPreferences and working notes for this workspace.", kind: "mission", want: false},
		{name: "placeholder in notes", content: "# Notes\n\n{todo}", kind: "notes", want: true},
		{name: "brace in real content", content: "Use {curly braces} in code", kind: "notes", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTemplate(tc.content, tc.kind); got != tc.want {
				t.Errorf("IsTemplate(%q, %q) = %v, want %v", tc.content, tc.kind, got, tc.want)
			}
		})
	}
}

func TestStripH1(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "has h1 with body", in: "# Title\n\nBody text", want: "Body text"},
		{name: "h1 only", in: "# Title", want: ""},
		{name: "no h1", in: "Just body", want: "Just body"},
		{name: "h2 not stripped", in: "## Subtitle\nBody", want: "## Subtitle\nBody"},
		{name: "empty", in: "", want: ""},
		{name: "h1 then h2", in: "# Title\n## Sub\nBody", want: "## Sub\nBody"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripH1(tc.in); got != tc.want {
				t.Errorf("StripH1(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
