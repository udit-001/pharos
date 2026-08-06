package extract

import (
	"fmt"
	"os"
)

// extractRTF extracts plain text from an RTF document using a brace-depth
// scanner (stdlib only): skips destination groups (fonttbl, info, pict, …),
// decodes \u{NNNN} unicode escapes and \'hh hex escapes, and emits escaped
// braces/backslashes literally.
func extractRTF(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	return rtfToText(data), "rtf-stdlib", nil
}

// rtfToText walks RTF bytes producing plain text. skipDepth is non-negative
// while inside a destination group marked \* (destinations to ignore).
func rtfToText(b []byte) string {
	out := make([]rune, 0, len(b))
	depth := 0
	skipDepth := -1

	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == '{':
			depth++
			if skipDepth < 0 && i+2 < len(b) && b[i+1] == '\\' && b[i+2] == '*' {
				skipDepth = depth
			}
			i++
		case c == '}':
			depth--
			if skipDepth == depth {
				skipDepth = -1
			}
			i++
		case c == '\\':
			nc := byte(0)
			if i+1 < len(b) {
				nc = b[i+1]
			}
			switch {
			case nc == '{' || nc == '}' || nc == '\\':
				if skipDepth < 0 {
					out = append(out, rune(nc))
				}
				i += 2
			case nc == '\'':
				if skipDepth < 0 && i+3 < len(b) && isHex(b[i+2]) && isHex(b[i+3]) {
					out = append(out, rune(hexVal(b[i+2])*16+hexVal(b[i+3])))
				}
				i += 4
			case nc == 'u':
				i = rtfUnicode(b, i, &out, skipDepth < 0)
			default:
				i = skipRTFControl(b, i)
			}
		case c == '\n' || c == '\r' || c == '\t':
			if skipDepth < 0 && len(out) > 0 && out[len(out)-1] != ' ' {
				out = append(out, ' ')
			}
			i++
		default:
			if skipDepth < 0 {
				out = append(out, rune(c))
			}
			i++
		}
	}
	return string(out)
}

// rtfUnicode handles a \u{NNNN} escape (optionally signed), emitting the
// codepoint when emit is true, then consumes the trailing fallback \'hh.
func rtfUnicode(b []byte, i int, out *[]rune, emit bool) int {
	j := i + 2
	neg := false
	if j < len(b) && b[j] == '-' {
		neg = true
		j++
	}
	n := 0
	for j < len(b) && b[j] >= '0' && b[j] <= '9' {
		n = n*10 + int(b[j]-'0')
		j++
	}
	if neg {
		n = -n
	}
	if emit && n != 0 {
		*out = append(*out, rune(uint16(n)))
	}
	// consume a following fallback \'hh if present
	if j+2 < len(b) && b[j] == '\\' && b[j+1] == '\'' {
		j += 4
	}
	return j
}

// skipRTFControl skips a control word and its optional parameter.
func skipRTFControl(b []byte, i int) int {
	j := i + 1
	for j < len(b) && isAlpha(b[j]) {
		j++
	}
	// optional sign+digits parameter, then a single following space
	if j < len(b) && (b[j] == '-' || b[j] == '+' || (b[j] >= '0' && b[j] <= '9')) {
		j++
		for j < len(b) && ((b[j] >= '0' && b[j] <= '9') || b[j] == '-') {
			j++
		}
	}
	if j < len(b) && b[j] == ' ' {
		j++
	}
	return j
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	default:
		return int(c-'A') + 10
	}
}
