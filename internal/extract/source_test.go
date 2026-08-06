package extract

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// TestPDFPartialGoLibFallback is the regression for the lebo102.pdf bug: a
// PDF the pure-Go lib can only partially read — one page chokes its lexer,
// the other yields text — must be surfaced as failedPages>0 and, when
// poppler is present, rescued by the pdftotext fallback. Before the fix the
// non-empty partial text was treated as a clean extraction and the fallback
// never fired.
func TestPDFPartialGoLibFallback(t *testing.T) {
	path := writeTemp(t, "partial.pdf", genPartialPDF())

	text, _, pages, failed, err := pdfGoLib(path)
	if err != nil {
		t.Fatalf("pdfGoLib: %v", err)
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2", pages)
	}
	if failed != 1 {
		t.Errorf("failedPages = %d, want 1 (page 2 must fail the go-lib)", failed)
	}
	if !textual(text) {
		t.Errorf("go-lib text unexpectedly empty — partial extraction not reproduced: %q", text)
	}
	if !strings.Contains(text, "GOOD PAGE ONE") {
		t.Errorf("go-lib text missing page 1 content: %q", text)
	}

	out, method, pages, err := extractPDF(path)
	if err != nil {
		t.Fatalf("extractPDF: %v", err)
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		// poppler absent: fallback stays optional, the go-lib method is kept.
		if method != "pdf" {
			t.Errorf("method = %q, want pdf (no poppler on PATH)", method)
		}
		t.Skip("pdftotext not on PATH — poppler method assertion skipped")
	}
	if method != "pdf-pdftotext" {
		t.Errorf("extractPDF method = %q, want pdf-pdftotext (fallback must fire)", method)
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2 (poppler rescues the failed page)", pages)
	}
	if !strings.Contains(out, "GOOD PAGE ONE") {
		t.Errorf("extractPDF text missing content: %q", out)
	}
}

// genPartialPDF builds a two-page PDF whose second page carries an invalid
// escape (\q) inside a content-stream literal string. The go-lib's lexer
// panics on it and GetPlainText turns that into a per-page error, so the
// go-lib extraction is non-empty but skips a page — the exact shape of the
// lebo102.pdf regression. Poppler tolerates the escape, so pdftotext
// recovers both pages.
func genPartialPDF() string {
	obj := func(n int, body string) string {
		return strconv.Itoa(n) + " 0 obj\n" + body + "\nendobj\n"
	}
	streamObj := func(n int, data string) string {
		return strconv.Itoa(n) + " 0 obj\n<< /Length " + strconv.Itoa(len(data)) + " >>\nstream\n" + data + "\nendstream\nendobj\n"
	}

	var out string
	out += "%PDF-1.4\n"
	offsets := make([]int, 0, 7)
	appendObj := func(s string) {
		offsets = append(offsets, len(out))
		out += s
	}
	appendObj(obj(1, "<< /Type /Catalog /Pages 2 0 R >>"))
	appendObj(obj(2, "<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>"))
	appendObj(obj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 7 0 R >> >> /Contents 4 0 R >>"))
	appendObj(streamObj(4, "BT /F1 12 Tf 72 720 Td (GOOD PAGE ONE) Tj ET"))
	appendObj(obj(5, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 7 0 R >> >> /Contents 6 0 R >>"))
	appendObj(streamObj(6, "BT /F1 12 Tf 72 700 Td (BAD \\q ESCAPE) Tj ET"))
	appendObj(obj(7, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"))

	xrefStart := len(out)
	out += "xref\n0 8\n0000000000 65535 f \n"
	for _, off := range offsets {
		out += fmt.Sprintf("%010d 00000 n \n", off)
	}
	out += "trailer\n<< /Size 8 /Root 1 0 R >>\nstartxref\n" + strconv.Itoa(xrefStart) + "\n%%EOF\n"
	return out
}

// TestPrefersPoppler pins the fallback decision: any non-empty extraction that
// parsed every page must NOT trigger the fallback, while empty text or any
// skipped page must. The second case is the regression that let partial
// go-lib text block the poppler rescue.
func TestPrefersPoppler(t *testing.T) {
	cases := []struct {
		name        string
		text        string
		failedPages int
		want        bool
	}{
		{"full extraction", "some real content\nwith a second line", 0, false},
		{"empty text", "", 0, true},
		{"whitespace text", "   \n  ", 0, true},
		{"one skipped page", "partial content", 1, true},
		{"most pages skipped", "snippets", 14, true},
	}
	for _, tc := range cases {
		if got := prefersPoppler(tc.text, tc.failedPages); got != tc.want {
			t.Errorf("%s: prefersPoppler(%q, %d) = %v, want %v", tc.name, tc.text, tc.failedPages, got, tc.want)
		}
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
