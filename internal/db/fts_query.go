package db

import "strings"

func buildFTSQuery(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, tok := range strings.Fields(s) {
		tok = strings.TrimRight(tok, "*")
		if tok == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(tok)
		b.WriteByte('*')
	}
	return b.String()
}
