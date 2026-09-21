package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/udit-001/pharos/internal/db"
)

// Authoring lint: static HTML checks over lesson/reference bodies, printed as
// warnings before the entity is saved. Warnings never block — a lesson that
// fails a rule is still created; --force silences the output entirely.
//
// The checks mirror the server's injection detection (internal/server/detect.go):
// a rule here is a construct the injector or dashboard will misinterpret —
// caught at authoring time instead of failing silently at render time.

// lintFrameBody returns human-readable warnings for the submitted body HTML.
// wsStore resolves asset references for the offline rule; nil skips that rule.
func lintFrameBody(wsStore *db.WorkspaceStore, data []byte) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}
	var warns []string
	warn := func(format string, args ...any) {
		warns = append(warns, "warning: "+fmt.Sprintf(format, args...))
	}

	// Quiz structure: .q needs .options and .fb, and data-answer must match
	// exactly one option button — the binder compares button text, so a
	// mismatched answer fails silently at render time.
	doc.Find(".q").Each(func(i int, q *goquery.Selection) {
		if q.Find(".options").Length() == 0 {
			warn(`quiz block #%d: missing .options wrapper`, i+1)
		}
		if q.Find(".fb").Length() == 0 {
			warn(`quiz block #%d: missing .fb feedback element`, i+1)
		}
		answer, has := q.Attr("data-answer")
		if !has {
			warn(`quiz block #%d: no data-answer attribute`, i+1)
			return
		}
		matches := 0
		q.Find(".options button").Each(func(_ int, b *goquery.Selection) {
			if strings.TrimSpace(b.Text()) == answer {
				matches++
			}
		})
		if matches == 0 {
			warn("quiz block #%d: data-answer %q matches no option button (exact text match)", i+1, answer)
		} else if matches > 1 {
			warn("quiz block #%d: data-answer %q matches %d option buttons — ambiguous", i+1, answer, matches)
		}
	})

	// Vega pairing: every container needs its spec and vice versa.
	vegaContainers := map[string]bool{}
	doc.Find("[data-vega]").Each(func(_ int, s *goquery.Selection) {
		id, ok := s.Attr("data-vega")
		if !ok || id == "" {
			warn("chart container: data-vega attribute is empty")
			return
		}
		vegaContainers[id] = true
		if doc.Find(`script[type="application/json"]#`+id).Length() == 0 {
			warn("chart %q: no <script type=\"application/json\" id=\"%s\"> spec", id, id)
		}
	})
	doc.Find(`script[type="application/json"]`).Each(func(_ int, s *goquery.Selection) {
		id, ok := s.Attr("id")
		if !ok || id == "" {
			return
		}
		if !vegaContainers[id] && doc.Find(`[data-vega="`+id+`"]`).Length() == 0 {
			warn("chart spec %q: no matching [data-vega] container", id)
		}
	})

	// Link safety: no parent-climbing asset paths (iframe escape 404s);
	// dashboard routes must escape with target="_top"; external anchors must
	// carry rel=noopener.
	doc.Find("[href], [src]").Each(func(_ int, s *goquery.Selection) {
		val := ""
		for _, attr := range []string{"href", "src"} {
			if v, ok := s.Attr(attr); ok {
				val = v
				break
			}
		}
		if strings.HasPrefix(val, "../") {
			tag := goquery.NodeName(s)
			warn("%s %q: parent-relative path (../) escapes the iframe root and 404s — use root-relative assets/", tag, val)
		}
	})

	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if strings.HasPrefix(href, "/workspace/") || href == "/about" {
			if target, _ := a.Attr("target"); target != "_top" {
				warn("link %q: dashboard route needs target=\"_top\" to escape the iframe", href)
			}
			return
		}
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			rel, _ := a.Attr("rel")
			if !strings.Contains(rel, "noopener") {
				warn("external link %q: needs rel=\"noopener noreferrer\"", href)
			}
		}
	})

	// Offline rule: CDN references break offline lessons; referenced assets
	// must exist on disk.
	doc.Find("script[src], link[href], img[src]").Each(func(_ int, s *goquery.Selection) {
		val := ""
		if v, ok := s.Attr("src"); ok {
			val = v
		} else if v, ok := s.Attr("href"); ok {
			val = v
		}
		if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
			warn("%s %q: CDN reference — vendored libs and fonts must come from assets/ (offline rule)", goquery.NodeName(s), val)
			return
		}
		if wsStore != nil && strings.HasPrefix(val, "assets/") {
			path, err := wsStore.AssetPath(strings.TrimPrefix(val, "assets/"))
			if err != nil {
				warn("%s %q: asset path escapes assets/", goquery.NodeName(s), val)
			} else if _, statErr := os.Stat(path); statErr != nil {
				warn("%s %q: referenced asset does not exist", goquery.NodeName(s), val)
			}
		}
	})

	// Glossary: tooltip spans carry the term they preview.
	doc.Find(".glossary-term").Each(func(i int, s *goquery.Selection) {
		if term, ok := s.Attr("data-term"); !ok || term == "" {
			warn("glossary span #%d: missing data-term attribute", i+1)
		}
	})

	sort.Strings(warns)
	return warns
}

// printLintWarnings prints lint warnings to stderr. Returns true when any
// warning was printed (used by --force to skip).
func printLintWarnings(warns []string) bool {
	if len(warns) == 0 {
		return false
	}
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "  "+w)
	}
	fmt.Fprintf(os.Stderr, "  (%d lint warning%s — pass --force to silence)\n", len(warns), plural(len(warns)))
	return true
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
