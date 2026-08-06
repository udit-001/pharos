package extract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	// docxDocumentPath is the single content file in a DOCX archive.
	docxDocumentPath = "word/document.xml"
	// docxMaxTextBytes bounds the decompressed document.xml to guard against
	// zip-bomb expansion.
	docxMaxTextBytes = 256 << 20
)

// errDocxUnsafe is returned when a DOCX carries forbidden XML entity
// declarations (XXE / Billion-Laughs class).
var errDocxUnsafe = errors.New("docx contains forbidden XML DOCTYPE/ENTITY declaration")

// extractDocumentXML returns the raw bytes of word/document.xml from a DOCX
// (a ZIP container). Uses only stdlib archive/zip.
func extractDocumentXML(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open docx zip: %w", err)
	}
	for _, f := range zr.File {
		if f.Name != docxDocumentPath {
			continue
		}
		if f.UncompressedSize64 > docxMaxTextBytes {
			return nil, fmt.Errorf("docx %s too large", docxDocumentPath)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", docxDocumentPath, err)
		}
		body, err := io.ReadAll(io.LimitReader(rc, docxMaxTextBytes))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", docxDocumentPath, err)
		}
		return body, nil
	}
	return nil, fmt.Errorf("docx missing %s", docxDocumentPath)
}

// validateDocxXMLSafety rejects any XML document declaring a DOCTYPE or
// ENTITY — the XXE and Billion-Laughs attack class. Rejects before parsing.
func validateDocxXMLSafety(xmlData []byte) error {
	lower := strings.ToLower(string(xmlData))
	if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<!entity") {
		return errDocxUnsafe
	}
	return nil
}

// docxToText flattens word/document.xml into plain text, preserving paragraph
// breaks and separating table cells with tabs so headings don't concatenate
// with body text. Matches element Local names so the w: namespace prefix is
// handled without a namespace resolver.
func docxToText(xmlData []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	var b strings.Builder
	inParagraph := false
	inText := false
	col := -1

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "tr":
				col = -1
			case "tc":
				col++
				if col > 0 {
					b.WriteByte('\t')
				}
			case "t":
				inText = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					b.WriteByte('\n')
					inParagraph = false
				}
			case "tr":
				if col >= 0 {
					b.WriteByte('\n')
					col = -1
				}
			case "t":
				inText = false
			}
		case xml.CharData:
			if inParagraph && inText {
				b.Write(t)
			}
		}
	}
	return b.String(), nil
}

// extractDOCX extracts text from a DOCX using only stdlib (archive/zip +
// encoding/xml). No external dependency. Returns the text and the method name.
func extractDOCX(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	xmlData, err := extractDocumentXML(data)
	if err != nil {
		return "", "", err
	}
	if err := validateDocxXMLSafety(xmlData); err != nil {
		return "", "", err
	}
	text, err := docxToText(xmlData)
	if err != nil {
		return "", "", fmt.Errorf("parse docx: %w", err)
	}
	return text, "docx-stdlib", nil
}
