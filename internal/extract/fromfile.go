package extract

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

// Result is the deterministic output of extracting a source document: the
// sanitized text plus the metadata the store persists and the agent consumes.
type Result struct {
	Format          Format    `json:"format"`
	Method          string    `json:"method"` // which extractor produced Text
	Text            string    `json:"text"`   // sanitized, structure-preserving
	Pages           int       `json:"pages"`  // page count where known (PDF/DOCX); 0 otherwise
	Chapters        []Chapter `json:"chapters"`
	EstimatedTokens int       `json:"estimatedTokens"`
}

// FromFile is the deep extraction seam: detect the format, run the matching
// faithful extractor, sanitize, estimate tokens, and detect chapters. Callers
// learn one function; the parser registry grows internally.
func FromFile(path string) (Result, error) {
	format := Detect(path)
	if format == FormatUnknown {
		return Result{}, fmt.Errorf("unsupported or unreadable format: %q", path)
	}

	var text, method string
	pages := 0
	var err error
	switch format {
	case FormatText:
		text, method, err = extractText(path)
	case FormatMarkdown:
		text, method, err = extractMarkdown(path)
	case FormatHTML:
		text, method, err = extractHTML(path)
	case FormatPDF:
		text, method, pages, err = extractPDF(path)
	case FormatDOCX:
		text, method, err = extractDOCX(path)
	case FormatEPUB:
		text, method, err = extractEPUB(path)
	case FormatRTF:
		text, method, err = extractRTF(path)
	}
	if err != nil {
		return Result{}, fmt.Errorf("extract %s: %w", format, err)
	}

	text = Sanitize(text)
	if strings.TrimSpace(text) == "" {
		return Result{}, fmt.Errorf("no extractable text in %q", path)
	}

	res := Result{
		Format:          format,
		Method:          method,
		Text:            text,
		Pages:           pages,
		Chapters:        detectChapters(text),
		EstimatedTokens: EstimateTokens(text),
	}
	return res, nil
}

// extractText reads a plain-text file with a BOM-aware decode chain
// (UTF-32, UTF-16, then UTF-8 → CP-1252 → Latin-1). Nothing is lost for the
// common case; the fallbacks rescue mislabeled 8-bit files.
func extractText(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	return decodeText(data), "text", nil
}

// extractMarkdown converts markdown to structure-preserving plain text:
// headings keep their line, code blocks (fenced) stay verbatim, and blank
// lines survive so chapter detection sees the structure. This is a faithful
// extractor for the source path — deliberately NOT the FTS-shaped
// FromMarkdown, which strips code.
func extractMarkdown(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	src := decodeText(data)
	var out []string
	inCode := false
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
		}
		if inCode {
			out = append(out, trimmed)
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), "faithful-markdown", nil
}

// FromHTMLFaithful converts HTML to plain text while keeping block elements on
// separate lines so headings/paragraphs don't concatenate (the quirk the
// FTS-shaped FromHTML has). Faithful for the source path; shared by the HTML
// and EPUB extractors.
func FromHTMLFaithful(html string) string {
	for _, tag := range []string{
		"</p>", "</h1>", "</h2>", "</h3>", "</h4>", "</h5>", "</h6>",
		"</div>", "</li>", "</ul>", "</ol>", "</table>", "</tr>", "</td>",
		"</blockquote>", "<br>", "<br/>", "<br />",
	} {
		html = strings.ReplaceAll(html, tag, "\n")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	doc.Find("head, script, style, noscript").Remove()
	return strings.TrimSpace(doc.Text())
}

// extractHTML converts HTML to plain text while keeping block elements on
// separate lines so headings/paragraphs don't concatenate (the quirk the
// FTS-shaped FromHTML has). Faithful for the source path.
func extractHTML(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	return FromHTMLFaithful(decodeText(data)), "faithful-html", nil
}

// decodeText decodes bytes into a UTF-8 string, preferring BOM-flagged UTF-32
// and UTF-16, then UTF-8, and finally the 8-bit sqlx / latin fallbacks.
func decodeText(data []byte) string {
	if len(data) >= 4 {
		switch {
		case data[0] == 0xFF && data[1] == 0xFE && data[2] == 0 && data[3] == 0:
			return decodeUTF32(data[4:], true)
		case data[0] == 0 && data[1] == 0 && data[2] == 0xFE && data[3] == 0xFF:
			return decodeUTF32(data[4:], false)
		}
	}
	if len(data) >= 2 {
		switch {
		case data[0] == 0xFF && data[1] == 0xFE:
			return decodeUTF16(data[2:], true)
		case data[0] == 0xFE && data[1] == 0xFF:
			return decodeUTF16(data[2:], false)
		}
	}
	if utf8.Valid(data) {
		return string(data)
	}
	// cp1252-ish fallback: treat each byte as a code point (Latin-1).
	return decodeLatin1(data)
}

func decodeUTF32(data []byte, littleEndian bool) string {
	n := len(data) / 4
	runes := make([]rune, 0, n)
	for i := 0; i+4 <= len(data); i += 4 {
		var u uint32
		if littleEndian {
			u = uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16 | uint32(data[i+3])<<24
		} else {
			u = uint32(data[i])<<24 | uint32(data[i+1])<<16 | uint32(data[i+2])<<8 | uint32(data[i+3])
		}
		runes = append(runes, rune(u))
	}
	return string(runes)
}

func decodeUTF16(data []byte, littleEndian bool) string {
	n := len(data) / 2
	units := make([]uint16, 0, n)
	for i := 0; i+2 <= len(data); i += 2 {
		var u uint16
		if littleEndian {
			u = uint16(data[i]) | uint16(data[i+1])<<8
		} else {
			u = uint16(data[i])<<8 | uint16(data[i+1])
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}

func decodeLatin1(data []byte) string {
	b := make([]rune, 0, len(data))
	for _, x := range data {
		b = append(b, rune(x))
	}
	return string(b)
}
