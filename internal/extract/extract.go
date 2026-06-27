package extract

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FromHTML parses HTML and returns the plain text content, stripping all
// markup. Used to index lesson/reference body content for full-text search.
func FromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	doc.Find("head, script, style, noscript").Remove()
	return strings.TrimSpace(doc.Text())
}

// FromMarkdown strips markdown formatting and returns plain text for FTS
// indexing.
//
// NOTE: This is an FTS indexing function, not a faithful text extractor.
// Code blocks (fenced and indented) are deliberately skipped because code
// tokens add noise to search results rather than signal. This coupling
// between "extract text" and "shape text for search" is named here so a
// future goldmark-AST rewrite can decide whether to split them (extract
// returns all text, an FTS filter strips code) or keep them fused.
func FromMarkdown(md string) string {
	lines := strings.Split(md, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Strip indented code blocks (4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			if trimmed == "" || strings.HasPrefix(trimmed, "```") {
				continue
			}
			// Not strictly a code block, keep the text but trimmed
		}

		// Strip blockquote and heading markers
		processed := strings.TrimLeft(line, " >")
		processed = strings.TrimLeft(processed, "#")
		processed = strings.TrimLeft(processed, " ")

		// Strip list markers: -, *, +, 1.
		if len(processed) > 0 {
			c := processed[0]
			if c == '-' || c == '+' || c == '*' {
				if len(processed) == 1 || processed[1] == ' ' {
					processed = strings.TrimLeft(processed[1:], " ")
				}
			} else if c >= '0' && c <= '9' {
				if idx := strings.Index(processed, ". "); idx > 0 && idx < 4 {
					processed = processed[idx+2:]
				} else if idx := strings.Index(processed, ") "); idx > 0 && idx < 4 {
					processed = processed[idx+2:]
				}
			}
		}

		// Skip horizontal rules
		if isHorizontalRule(processed) {
			continue
		}

		// Strip remaining inline formatting: keep text from links, strip markers
		processed = stripInlineMarkdown(processed)

		processed = strings.TrimSpace(processed)
		if processed != "" {
			result = append(result, processed)
		}
	}

	return strings.TrimSpace(strings.Join(result, " "))
}

func isHorizontalRule(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return false
	}
	for _, r := range s {
		if r != '-' && r != '*' && r != '_' {
			return false
		}
	}
	return true
}

func stripInlineMarkdown(s string) string {
	var b strings.Builder
	i := 0
	runes := []rune(s)

	for i < len(runes) {
		c := runes[i]

		// Skip image ![alt](url) — keep alt text
		if c == '!' && i+1 < len(runes) && runes[i+1] == '[' {
			end := findMatchingBracket(runes, i+1, '[', ']')
			if end < 0 {
				b.WriteRune(c)
				i++
				continue
			}
			altText := runes[i+2 : end]
			// Skip the URL part
			if end+1 < len(runes) && runes[end+1] == '(' {
				parenEnd := findMatchingParen(runes, end+1)
				if parenEnd >= 0 {
					// Recurse on alt text (may contain formatting)
					b.WriteString(stripInlineMarkdown(string(altText)))
					b.WriteRune(' ')
					i = parenEnd + 1
					continue
				}
			}
			b.WriteString(stripInlineMarkdown(string(altText)))
			b.WriteRune(' ')
			i = end + 1
			continue
		}

		// Link [text](url) — keep text
		if c == '[' {
			end := findMatchingBracket(runes, i, '[', ']')
			if end < 0 {
				b.WriteRune(c)
				i++
				continue
			}
			text := runes[i+1 : end]
			if end+1 < len(runes) && runes[end+1] == '(' {
				parenEnd := findMatchingParen(runes, end+1)
				if parenEnd >= 0 {
					b.WriteString(stripInlineMarkdown(string(text)))
					b.WriteRune(' ')
					i = parenEnd + 1
					continue
				}
			}
			b.WriteString(stripInlineMarkdown(string(text)))
			i = end + 1
			continue
		}

		// Skip HTML tags
		if c == '<' {
			gt := strings.IndexRune(string(runes[i:]), '>')
			if gt >= 0 {
				i += gt + 1
				continue
			}
		}

		// Skip backtick code spans — keep the text inside
		if c == '`' {
			end := i + 1
			for end < len(runes) && runes[end] == '`' {
				end++
			}
			backtickLen := end - i
			closing := findBacktickSequence(runes, end, backtickLen)
			if closing >= 0 {
				b.WriteString(string(runes[end:closing]))
				b.WriteRune(' ')
				i = closing + backtickLen
				continue
			}
		}

		b.WriteRune(c)
		i++
	}

	return b.String()
}

func findMatchingBracket(runes []rune, start int, open, close rune) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if runes[i] == open {
			depth++
		} else if runes[i] == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingParen(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if runes[i] == '(' {
			depth++
		} else if runes[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findBacktickSequence(runes []rune, start int, n int) int {
	for i := start; i <= len(runes)-n; i++ {
		match := true
		for j := 0; j < n; j++ {
			if runes[i+j] != '`' {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
