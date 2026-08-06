package extract

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractPDF extracts text from a text-layer PDF. The zero-dependency path is
// the pure-Go ledongthuc/pdf library; when that yields no usable text, or
// skips pages it could not parse (the Go lib reads some PDFs poorly), it
// falls back to the optional poppler `pdftotext` binary when present on PATH
// (LookPath — never a build or hard runtime dependency). PDF ingestion
// therefore works out of the box anywhere, while tricky PDFs are rescued where
// poppler is installed.
func extractPDF(path string) (string, string, int, error) {
	text, method, pages, failedPages, err := pdfGoLib(path)
	if err != nil {
		return "", "", 0, err
	}
	if prefersPoppler(text, failedPages) {
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

// prefersPoppler reports whether the go-lib extraction is untrustworthy
// enough to prefer the poppler fallback: no usable text at all, or any page
// the go-lib failed to parse (a silent skip leaves the extraction
// incomplete — partial text must not block the fallback).
func prefersPoppler(text string, failedPages int) bool {
	return !textual(text) || failedPages > 0
}

// pdfGoLib extracts via the pure-Go ledongthuc/pdf library (no system deps).
// Pages that are null or fail to parse are skipped and counted in the returned
// failedPages — a partial extraction is a real signal for the caller, not a
// successful one. A zero-text result is valid (the caller tries the poppler
// fallback) and is not an error.
func pdfGoLib(path string) (text string, method string, pages int, failedPages int, err error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("open pdf %s: %w", path, err)
	}
	defer f.Close()
	pages = r.NumPage()
	parts := make([]string, 0, pages)
	for i := 1; i <= pages; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			failedPages++
			continue
		}
		t, err := p.GetPlainText(nil)
		if err != nil {
			failedPages++
			continue
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, "\n"), "pdf", pages, failedPages, nil
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
