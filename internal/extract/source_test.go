package extract

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestDetect(t *testing.T) {
	cases := map[string]Format{
		"a.txt":     FormatText,
		"b.md":      FormatMarkdown,
		"c.html":    FormatHTML,
		"d.pdf":     FormatPDF,
		"e.docx":    FormatDOCX,
		"f.epub":    FormatEPUB,
		"g.rtf":     FormatRTF,
		"noext.txt": FormatText,
	}
	for name, want := range cases {
		if got := Detect(writeTemp(t, name, "x")); got != want {
			t.Errorf("Detect(%s) = %s, want %s", name, got, want)
		}
	}
}

func TestDetectMagicBytes(t *testing.T) {
	if got := Detect(writeTemp(t, "foo.bin", "%PDF-1.7")); got != FormatPDF {
		t.Errorf("magic PDF detect = %s, want pdf", got)
	}
	if got := Detect(writeTemp(t, "foo.bin", "{\\rtf1")); got != FormatRTF {
		t.Errorf("magic RTF detect = %s, want rtf", got)
	}
}

func TestSanitize(t *testing.T) {
	in := "hel\u200Blo\u202Eworld\uFEFF \U000E0001 tag \u3164"
	if got := Sanitize(in); strings.ContainsAny(got, "\u200b\u202e\ufeff") {
		t.Errorf("Sanitize left invisible controls in %q", got)
	}
	if !strings.Contains(Sanitize("normal text"), "normal text") {
		t.Error("Sanitize dropped normal text")
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens("one two three"); got < 3 {
		t.Errorf("latin tokens = %d, want >= 3", got)
	}
	cjk := "学习工具文档"
	if got := EstimateTokens(cjk); got < 3 {
		t.Errorf("cjk tokens = %d, want >= 3", got)
	}
}

func TestFromFileText(t *testing.T) {
	p := writeTemp(t, "book.txt", "Chapter 1: Joins\nIntro text here.\n\nChapter 2: Indexes\nMore content.\n")
	res, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if res.Format != FormatText {
		t.Errorf("format = %s, want text", res.Format)
	}
	if len(res.Chapters) != 2 {
		t.Errorf("chapters = %d, want 2", len(res.Chapters))
	}
	if res.EstimatedTokens == 0 {
		t.Error("expected tokens > 0")
	}
}

func TestFromFileSanitizes(t *testing.T) {
	p := writeTemp(t, "evil.txt", "safe\u202Ebad\u200B")
	res, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if strings.Contains(res.Text, "\u202e") || strings.Contains(res.Text, "\u200b") {
		t.Errorf("text not sanitized: %q", res.Text)
	}
	if !strings.Contains(res.Text, "safe") {
		t.Error("safe text lost")
	}
}

func TestFromFileMarkdownKeepsCode(t *testing.T) {
	p := writeTemp(t, "notes.md", "# Title\n\n```sql\nSELECT 1;\n```\n\nBody.\n")
	res, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if !strings.Contains(res.Text, "SELECT 1;") {
		t.Errorf("code block not preserved: %q", res.Text)
	}
}

func TestFromFileHTMLNoConcatenation(t *testing.T) {
	p := writeTemp(t, "page.html", "<h1>Title</h1><p>One</p><p>Two</p>")
	res, err := FromFile(p)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	if strings.Contains(res.Text, "TitleOneTwo") {
		t.Errorf("blocks concatenated: %q", res.Text)
	}
}

func TestFromFilePDF(t *testing.T) {
	res, err := FromFile(filepath.Join("testdata", "sample.pdf"))
	if err != nil {
		t.Fatalf("FromFile sample.pdf: %v", err)
	}
	if res.Format != FormatPDF {
		t.Errorf("format = %s, want pdf", res.Format)
	}
	if res.Pages == 0 {
		t.Error("expected page count > 0")
	}
	if !strings.Contains(res.Text, "This is a heading") {
		t.Errorf("pdf text missing known content: %q", res.Text)
	}
}

// TestPDFPopplerFallback exercises the optional pdftotext fallback path when
// poppler is installed; skipped otherwise (fallback stays truly optional).
func TestPDFPopplerFallback(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not on PATH — fallback untestable here")
	}
	out, err := pdftotextText(filepath.Join("testdata", "sample.pdf"))
	if err != nil {
		t.Fatalf("pdftotext fallback: %v", err)
	}
	if !strings.Contains(out, "This is a heading") {
		t.Errorf("pdftotext fallback text missing known content: %q", out)
	}
	if n, err := pdfinfoPages(filepath.Join("testdata", "sample.pdf")); err != nil || n == 0 {
		t.Errorf("pdfinfo page count = %d (err %v), want > 0", n, err)
	}
}

func TestFromFileDOCX(t *testing.T) {
	res, err := FromFile(filepath.Join("testdata", "sample.docx"))
	if err != nil {
		t.Fatalf("FromFile sample.docx: %v", err)
	}
	if res.Format != FormatDOCX {
		t.Errorf("format = %s, want docx", res.Format)
	}
	for _, want := range []string{"Joins", "Joins combine rows", "alpha", "beta"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("docx text missing %q: %q", want, res.Text)
		}
	}
	// Heading and body must not have concatenated.
	if strings.Contains(res.Text, "JoinsJoins") {
		t.Errorf("docx paragraph concatenated: %q", res.Text)
	}
}

func TestFromFileUnsupportedUnknownFormat(t *testing.T) {
	p := writeTemp(t, "bogus.xyz", "no known format")
	if _, err := FromFile(p); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestFromFileEPUB(t *testing.T) {
	res, err := FromFile(filepath.Join("testdata", "sample.epub"))
	if err != nil {
		t.Fatalf("FromFile sample.epub: %v", err)
	}
	if res.Format != FormatEPUB {
		t.Errorf("format = %s, want epub", res.Format)
	}
	for _, want := range []string{"Photosynthesis", "Chlorophyll", "Respiration", "Mitochondria"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("epub text missing %q: %q", want, res.Text)
		}
	}
}

func TestFromFileRTF(t *testing.T) {
	res, err := FromFile(filepath.Join("testdata", "sample.rtf"))
	if err != nil {
		t.Fatalf("FromFile sample.rtf: %v", err)
	}
	if res.Format != FormatRTF {
		t.Errorf("format = %s, want rtf", res.Format)
	}
	for _, want := range []string{"Photosynthesis", "Chlorophyll", "Mitochondria", "energy"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("rtf text missing %q: %q", want, res.Text)
		}
	}
}

// TestDocxRejectsEntityDeclarations proves the XXE / Billion-Laughs gate: a
// document.xml declaring a DOCTYPE/ENTITY must be refused at parse time — and
// the real reason must propagate (not be swallowed into "no extractable text").
func TestDocxRejectsEntityDeclarations(t *testing.T) {
	// Build a DOCX whose document.xml is hostile by hand.
	evil := `<?xml version="1.0"?><!DOCTYPE lolz [ <!ENTITY laugh "lol"> ]><w:document><w:body><w:p><w:r><w:t>&laugh;</w:t></w:r></w:p></w:body></w:document>`
	path := filepath.Join(t.TempDir(), "evil.docx")
	mkDocxZip(t, path, evil)

	if _, _, err := extractDOCX(path); !errors.Is(err, errDocxUnsafe) {
		t.Fatalf("extractDOCX: expected errDocxUnsafe, got %v", err)
	}
	// FromFile must surface the rejection, not the vague empty-text error.
	_, err := FromFile(path)
	if err == nil || !strings.Contains(err.Error(), "DOCTYPE") && !errors.Is(err, errDocxUnsafe) {
		t.Fatalf("FromFile: expected the XXE rejection to propagate, got %v", err)
	}
}

func TestFromFileEmptyText(t *testing.T) {
	p := writeTemp(t, "empty.txt", "\u200b\u200b")
	if _, err := FromFile(p); err == nil {
		t.Error("expected error for empty sanitized source")
	}
}

// mkDocxZip writes a minimal DOCX archive around document.xml content.
func mkDocxZip(t *testing.T, path, documentXML string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
