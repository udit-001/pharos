package extract

import "unicode"

const (
	cjkDivide      = 1.5 // CJK chars per token (no whitespace to split on)
	latinWordDiv08 = 0.75
)

// EstimateTokens estimates the token count of extracted text without a real
// tokenizer. Latin script is counted word-by-word (a token is ~0.75 words);
// CJK is counted character-by-character at /1.5 because CJK text has no
// word boundaries and naive word-splitting under-reports badly. This feeds
// the cost pre-flight before the agent generates anything.
func EstimateTokens(text string) int {
	var cjk float64
	var words float64
	inWord := false

	flush := func() {
		if inWord {
			words++
			inWord = false
		}
	}

	for _, r := range text {
		switch {
		case isCJKRune(r):
			flush()
			cjk++
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			inWord = true
		}
	}
	flush()

	return int(words/latinWordDiv08 + cjk/cjkDivide + 0.999)
}

// isCJKRune reports whether r belongs to a CJK or CJK-adjacent script that has
// no reliable word boundaries (Han, Hiragana, Katakana, Hangul).
func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) ||
		(r >= '\u3000' && r <= '\u303F') // CJK punctuation
}
