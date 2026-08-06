package extract

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractPDF extracts text from a text-layer PDF. The zero-dependency path is
// the pure-Go ledongthuc/pdf library; when that yields no usable text (an
// empty result, or a PDF the Go lib reads poorly), it falls back to the
// optional poppler `pdftotext` binary when present on PATH (LookPath — never a
// build or hard runtime dependency). PDF ingestion therefore works out of the
// box anywhere, while tricky PDFs are rescued where poppler is installed.
func extractPDF(path string) (string, string, int, error) {
	text, method, pages, err := pdfGoLib(path)
	if err != nil {
		return "", "", 0, err
	}
	if !textual(text) {
		if _, err := exec.LookPath("pdftotext"); err == nil {
			if out, err := pdftotextText(path); err == nil && textual(out) {
				text, method = out, "pdf-pdftotext"
				if _, err := exec.LookPath("pdfinfo"); err == nil {
					if n, err := pdfinfoPages(path); err == nil && n > 0 {
						pages = n
					}
				}
			}
		}
	}
	return text, method, pages, nil
}

// pdfGoLib extracts via the pure-Go ledongthuc/pdf library (no system deps).
// A zero-text result is valid (the caller tries the poppler fallback) and is
// not an error.
func pdfGoLib(path string) (string, string, int, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", "", 0, fmt.Errorf("open pdf %s: %w", path, err)
	}
	defer f.Close()
	pages := r.NumPage()
	parts := make([]string, 0, pages)
	for i := 1; i <= pages; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		t, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, "\n"), "pdf", pages, nil
}

// pdftotextText runs `pdftotext -layout <path> -` and returns stdout.
func pdftotextText(path string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// pdfinfoPages reads the page count from `pdfinfo <path>`.
func pdfinfoPages(path string) (int, error) {
	cmd := exec.Command("pdfinfo", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`(?m)^Pages:\s+(\d+)\s*$`)
	m := re.FindSubmatch(out)
	if len(m) < 2 {
		return 0, nil
	}
	return atoi(m[1]), nil
}

func atoi(b []byte) int {
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// textual reports whether extracted text has any usable content.
func textual(s string) bool {
	return strings.TrimSpace(s) != ""
}
