package extract

import "testing"

func TestFromHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple paragraph",
			input: `<html><body><p>Hello world</p></body></html>`,
			want:  "Hello world",
		},
		{
			name:  "strips head script style noscript",
			input: `<html><head><title>X</title><script>bad()</script><style>.x{}</style><noscript>nope</noscript></head><body><p>keep</p></body></html>`,
			want:  "keep",
		},
		{
			// Quirk: goquery .Text() concatenates block elements without spaces.
			name:  "multiple elements concatenate without spaces (quirk)",
			input: `<h1>Title</h1><p>One</p><p>Two</p>`,
			want:  "TitleOneTwo",
		},
		{
			name:  "empty html",
			input: ``,
			want:  "",
		},
		{
			name:  "malformed html returns what goquery can parse",
			input: `<p>unclosed`,
			want:  "unclosed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FromHTML(tc.input)
			if got != tc.want {
				t.Errorf("FromHTML(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFromMarkdown_BasicStructure(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text passes through",
			input: "Just some text.",
			want:  "Just some text.",
		},
		{
			name:  "heading marker stripped",
			input: "# Title\nbody",
			want:  "Title body",
		},
		{
			name:  "h2 and h3 markers stripped",
			input: "## Sub\n### Subsub\ncontent",
			want:  "Sub Subsub content",
		},
		{
			name:  "blockquote marker stripped",
			input: "> quoted line",
			want:  "quoted line",
		},
		{
			name:  "unordered list markers stripped",
			input: "- first\n* second\n+ third",
			want:  "first second third",
		},
		{
			name:  "ordered list markers stripped",
			input: "1. one\n2. two\n10. ten",
			want:  "one two ten",
		},
		{
			name:  "paren ordered list markers stripped",
			input: "1) one\n2) two",
			want:  "one two",
		},
		{
			name:  "horizontal rule skipped",
			input: "above\n---\nbelow",
			want:  "above below",
		},
		{
			name:  "horizontal rule asterisks skipped",
			input: "above\n***\nbelow",
			want:  "above below",
		},
		{
			name:  "horizontal rule underscores skipped",
			input: "above\n___\nbelow",
			want:  "above below",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \n\n  \n",
			want:  "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FromMarkdown(tc.input)
			if got != tc.want {
				t.Errorf("FromMarkdown(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFromMarkdown_CodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced code block skipped",
			input: "before\n```\nfunc main() {}\n```\nafter",
			want:  "before after",
		},
		{
			name:  "fenced code block with language skipped",
			input: "before\n```go\nfunc main() {}\n```\nafter",
			want:  "before after",
		},
		{
			// Quirk: indented lines that aren't blank/``` are kept (trimmed), not treated as code.
			name:  "indented non-empty text kept not dropped (quirk)",
			input: "before\n    code line\nafter",
			want:  "before code line after",
		},
		{
			name:  "tab-indented non-empty text kept (quirk)",
			input: "before\n\tcode line\nafter",
			want:  "before code line after",
		},
		{
			name:  "indented not-really-code kept (quirk)",
			input: "before\n    not really code\nafter",
			want:  "before not really code after",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FromMarkdown(tc.input)
			if got != tc.want {
				t.Errorf("FromMarkdown(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFromMarkdown_InlineFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// Quirk: link replacement writes a trailing space, leaving a double space with following text.
			name:  "link keeps text (double-space quirk)",
			input: "see [Pharos docs](https://example.com) here",
			want:  "see Pharos docs  here",
		},
		{
			name:  "image keeps alt text (double-space quirk)",
			input: "diagram ![Venn diagram](venn.png) shown",
			want:  "diagram Venn diagram  shown",
		},
		{
			name:  "backtick code span keeps inner text (double-space quirk)",
			input: "use `extract.FromHTML` now",
			want:  "use extract.FromHTML  now",
		},
		{
			name:  "html tag stripped (double-space quirk)",
			input: "line <br> break",
			want:  "line  break",
		},
		{
			name:  "html tag with attributes stripped",
			input: `text <a href="x">link</a> more`,
			want:  "text link more",
		},
		{
			// Quirk: emphasis markers (**, _, *) are NOT stripped — only links/images/tags/code-spans are handled.
			name:  "nested link keeps bold markers (quirk)",
			input: "see [**bold link**](url) here",
			want:  "see **bold link**  here",
		},
		{
			name:  "image alt keeps bold markers (quirk)",
			input: "![**bold alt**](x.png)",
			want:  "**bold alt**",
		},
		{
			name:  "unclosed link bracket kept literal",
			input: "see [unclosed here",
			want:  "see [unclosed here",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FromMarkdown(tc.input)
			if got != tc.want {
				t.Errorf("FromMarkdown(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFromMarkdown_RealisticRecord(t *testing.T) {
	input := `# What I learned about SQL joins

## Key insight
Joins combine rows from multiple tables on a shared column.

## Example
- INNER JOIN keeps matching rows only
- LEFT JOIN keeps all left rows

See [SQL joins](https://example.com/joins) for more.

` + "```sql" + `
SELECT * FROM a JOIN b ON a.id = b.id;
` + "```" + `

> Note: always alias your tables.
`
	want := "What I learned about SQL joins Key insight Joins combine rows from multiple tables on a shared column. Example INNER JOIN keeps matching rows only LEFT JOIN keeps all left rows See SQL joins  for more. Note: always alias your tables."
	got := FromMarkdown(input)
	if got != want {
		t.Errorf("FromMarkdown(realistic) mismatch:\n got: %q\nwant: %q", got, want)
	}
}
