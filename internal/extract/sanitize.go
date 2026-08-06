package extract

import "strings"

// Sanitize strips invisible Unicode that is hostile to an LLM context:
// zero-width spaces and joiners, bidi control characters (the Trojan-Source
// class, CVE-2021-42574), invisible Hangul fillers, and the entire Unicode
// tag block. Untrusted documents must pass through this before their text is
// stored or handed to the agent.
func Sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isInvisibleRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isInvisibleRune reports whether r is one of the invisible/control code
// points stripped by Sanitize.
func isInvisibleRune(r rune) bool {
	// Zero-width / format characters.
	if r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\uFEFF' {
		return true
	}
	// Bidi controls: explicit embedding/override/isolate + LRM/RLM/ALM.
	if (r >= '\u202A' && r <= '\u202E') || (r >= '\u2066' && r <= '\u2069') {
		return true
	}
	if r == '\u200E' || r == '\u200F' || r == '\u061C' {
		return true
	}
	// Invisible Hangul fillers.
	if r == '\u3164' || r == '\uFFA0' {
		return true
	}
	// Unicode tag block (U+E0000..U+E007F).
	if r >= '\U000E0000' && r <= '\U000E007F' {
		return true
	}
	return false
}
