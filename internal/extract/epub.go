package extract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
)

// average EPUB spine content sizes are small; these caps bound zip-bomb reads.
const (
	epubCapContainer = 1 << 20
	epubCapContent   = 8 << 20
)

// containerRootfileRe locates the OPF full-path inside META-INF/container.xml.
var containerRootfileRe = regexp.MustCompile(`<rootfile[^>]*full-path\s*=\s*"([^"]+)"`)

// extractEPUB extracts text from an EPUB (a ZIP container) using only stdlib:
// resolve the OPF via META-INF/container.xml, read the manifest + spine, then
// extract each spine content document (XHTML) in order and join them. Returns
// the text and the method name.
func extractEPUB(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", "", fmt.Errorf("open epub zip %s: %w", path, err)
	}

	opfPath := epubOPFPath(zr)
	if opfPath == "" {
		return "", "", errors.New("epub: no OPF in META-INF/container.xml")
	}
	dir, manifest, spine := epubManifestAndSpine(zr, opfPath)
	if len(spine) == 0 {
		return "", "", errors.New("epub: empty spine")
	}

	var parts []string
	for _, idref := range spine {
		href, ok := manifest[idref]
		if !ok || href == "" {
			continue
		}
		contentPath := normalizeEPUBPath(dir, href)
		body := readZIP(zr, contentPath, epubCapContent)
		if body == nil {
			continue
		}
		if t := FromHTMLFaithful(string(body)); strings.TrimSpace(t) != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 {
		return "", "", errors.New("epub: no content extracted from spine")
	}
	return strings.Join(parts, "\n"), "epub-stdlib", nil
}

// readZIP returns the decompressed bytes of the entry with the exact name, or
// nil when absent. Capped at limit.
func readZIP(zr *zip.Reader, name string, limit int64) []byte {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		b, _ := io.ReadAll(io.LimitReader(rc, limit))
		rc.Close()
		return b
	}
	return nil
}

// epubOPFPath returns the OPF path from META-INF/container.xml.
func epubOPFPath(zr *zip.Reader) string {
	container := readZIP(zr, "META-INF/container.xml", epubCapContainer)
	if container == nil {
		return ""
	}
	m := containerRootfileRe.FindSubmatch(container)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
}

// epubManifestAndSpine reads the OPF and returns its directory, the manifest
// (idref → href), and the ordered spine (list of idrefs).
func epubManifestAndSpine(zr *zip.Reader, opfPath string) (string, map[string]string, []string) {
	dec := xml.NewDecoder(bytes.NewReader(readZIP(zr, opfPath, epubCapContent)))
	manifest := map[string]string{}
	var spine []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "item":
			manifest[attr(se, "id")] = attr(se, "href")
		case "itemref":
			if idref := attr(se, "idref"); idref != "" {
				spine = append(spine, idref)
			}
		}
	}
	return path.Dir(opfPath), manifest, spine
}

// attr returns an element attribute value by Local name.
func attr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// normalizeEPUBPath joins the OPF directory with a spine href, handling a
// leading slash and dot segments.
func normalizeEPUBPath(dir, href string) string {
	href = strings.TrimPrefix(href, "/")
	if dir != "." && dir != "" {
		href = path.Join(dir, href)
	}
	return path.Clean(href)
}
