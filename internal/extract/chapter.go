package extract

import "strings"

// Chapter is a detected section boundary in an extracted document's text.
// Offset is the index into Text at which the chapter begins, in runes, so the
// store can slice the chapter out of the stored body cheaply.
type Chapter struct {
	Title  string `json:"title"`
	Offset int    `json:"offset"` // rune offset into Result.Text
}

// detectChapters scans faithful text for heading-like lines and returns the
// chapter boundaries. It is a focused heuristic (English/roman-numeral/
// numbered headings) rather than the full book-to-skill multilingual catalog;
// an empty result is valid and callers fall back to whole-text access.
func detectChapters(text string) []Chapter {
	var chapters []Chapter
	runeCount := 0
	for _, line := range strings.Split(text, "\n") {
		if isChapterHeading(line) {
			chapters = append(chapters, Chapter{Title: strings.TrimSpace(line), Offset: runeCount})
		}
		runeCount += len([]rune(line)) + 1 // +1 for the newline
	}
	return chapters
}

// isChapterHeading reports whether line looks like a chapter/section heading
// and not prose. A heading must end at end-of-line or lead with a capital /
// digit so that "Chapter 6 explores…" (lowercase continuation) is rejected.
func isChapterHeading(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	lower := strings.ToLower(t)
	if startsWithAny(lower, "chapter ", "chap. ", "chapter: ") {
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lower, "chapter: "), "chapter "))
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "chap. "))
		return headingOK(rest)
	}
	// Roman-numeral heading: "VI", "I: Loomings", "iv.".
	if isRomanHeading(t) {
		return true
	}
	// Numbered section: "N" then "."/":"/"-" then a capitalized leading word.
	if len(t) >= 2 && t[0] >= '0' && t[0] <= '9' {
		idx := strings.IndexAny(t, ".:-")
		if idx > 0 && idx < 5 {
			return headingOK(strings.TrimSpace(t[:idx]))
		}
	}
	return false
}

// headingOK accepts a numeric/roman tail ("1", "VI") or one beginning with a
// capital letter ("VI: Loomings", "3: Joins"), rejecting lowercase
// continuations that read as prose.
func headingOK(tail string) bool {
	if tail == "" {
		return true
	}
	first := tail[0]
	if first >= '0' && first <= '9' || first >= 'A' && first <= 'Z' {
		return true
	}
	return false
}

// isRomanHeading matches an all-roman-numeral tail optionally followed by a
// colon and title ("I: Loomings", "VI.", "iv: x").
func isRomanHeading(t string) bool {
	body := strings.TrimSpace(strings.TrimSuffix(t, "."))
	if i := colonIndex(body); i > 0 {
		body = strings.TrimSpace(body[:i])
	}
	if body == "" {
		return false
	}
	if !isRomanNumeral(body) {
		return false
	}
	return true
}

func isRomanNumeral(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch r {
		case 'i', 'v', 'x', 'l', 'c', 'd', 'm', 'I', 'V', 'X', 'L', 'C', 'D', 'M':
		default:
			return false
		}
	}
	return true
}

func colonIndex(s string) int {
	for i, r := range s {
		if r == ':' {
			return i
		}
	}
	return -1
}

func startsWithAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
