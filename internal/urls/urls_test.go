package urls

import "testing"

func TestPathEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no spaces", input: "alpha", want: "alpha"},
		{name: "single space", input: "go snippets", want: "go%20snippets"},
		{name: "multiple spaces", input: "a b c", want: "a%20b%20c"},
		{name: "leading space", input: " lead", want: "%20lead"},
		{name: "empty", input: "", want: ""},
		{name: "slash escaped", input: "bad/name", want: "bad%2Fname"},
		{name: "question mark escaped", input: "bad?name", want: "bad%3Fname"},
		{name: "hash escaped", input: "bad#name", want: "bad%23name"},
		{name: "percent escaped", input: "bad%name", want: "bad%25name"},
		{name: "plus is valid in paths (not escaped)", input: "C++", want: "C++"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PathEscape(tc.input); got != tc.want {
				t.Errorf("PathEscape(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuilders(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "workspace", got: Workspace("go snippets"), want: "/workspace/go%20snippets"},
		{name: "workspace no spaces", got: Workspace("alpha"), want: "/workspace/alpha"},
		{name: "lesson", got: Lesson("go snippets", 3), want: "/workspace/go%20snippets/lesson/3"},
		{name: "lesson seq 1", got: Lesson("alpha", 1), want: "/workspace/alpha/lesson/1"},
		{name: "record", got: Record("go snippets", 7), want: "/workspace/go%20snippets/record/7"},
		{name: "ref", got: Ref("go snippets", "joins"), want: "/workspace/go%20snippets/ref/joins"},
		{name: "ref slug escaped if it had spaces", got: Ref("alpha", "two words"), want: "/workspace/alpha/ref/two%20words"},
		{name: "doc mission", got: Doc("go snippets", "mission"), want: "/workspace/go%20snippets/mission"},
		{name: "doc resources", got: Doc("alpha", "resources"), want: "/workspace/alpha/resources"},
		{name: "doc notes", got: Doc("alpha", "notes"), want: "/workspace/alpha/notes"},
		{name: "doc glossary", got: Doc("alpha", "glossary"), want: "/workspace/alpha/glossary"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}
