package extract

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
)

// Format identifies the detected kind of a source document. The value is
// stored on the SourceDoc record and drives the parser dispatch in FromFile.
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatPDF      Format = "pdf"
	FormatDOCX     Format = "docx"
	FormatEPUB     Format = "epub"
	FormatRTF      Format = "rtf"
	FormatUnknown  Format = "unknown"
)

// extFormats maps file extensions (lower-cased, with dot) to a Format.
var extFormats = map[string]Format{
	".txt": FormatText, ".text": FormatText, ".org": FormatText,
	".md": FormatMarkdown, ".markdown": FormatMarkdown,
	".rst": FormatMarkdown, ".adoc": FormatMarkdown, ".asciidoc": FormatMarkdown,
	".html": FormatHTML, ".htm": FormatHTML, ".xhtml": FormatHTML,
	".pdf":  FormatPDF,
	".docx": FormatDOCX,
	".epub": FormatEPUB,
	".rtf":  FormatRTF,
}

// Detect identifies a source document's Format from its filename extension,
// falling back to magic-byte sniffing for extensionless or unsupported names.
// Magic bytes let a misnamed .bin that really holds a PDF still extract.
func Detect(path string) Format {
	if f, ok := extFormats[strings.ToLower(filepath.Ext(path))]; ok {
		return f
	}
	switch sniff(path) {
	case "%PDF":
		return FormatPDF
	case "PK\x03\x04":
		return pickZIPFormat(path)
	case "{\\rt":
		return FormatRTF
	}
	return FormatUnknown
}

// sniff reads up to 4 bytes and returns them as a string.
func sniff(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 4)
	n, _ := f.Read(buf)
	return string(buf[:n])
}

// pickZIPFormat distinguishes DOCX from EPUB by peeking at the zip central
// directory. EPUB locates its OPF via META-INF/container.xml; DOCX holds
// word/document.xml. Reading the entry names is cheap relative to a full
// extraction and keeps the sniff deterministic.
func pickZIPFormat(path string) Format {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return FormatUnknown
	}
	defer zr.Close()
	hasWord := false
	hasContainer := false
	for _, f := range zr.File {
		name := f.Name
		if name == "word/document.xml" {
			hasWord = true
		}
		if name == "META-INF/container.xml" {
			hasContainer = true
		}
	}
	switch {
	case hasWord:
		return FormatDOCX
	case hasContainer:
		return FormatEPUB
	default:
		return FormatUnknown
	}
}
