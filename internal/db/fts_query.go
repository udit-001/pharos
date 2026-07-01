package db

import "strings"

// bm25Weights is the ranking weight policy for all FTS tables: title weight,
// summary weight, body weight. One constant — changing it updates all three
// search methods at once.
const bm25Weights = "10.0, 5.0, 1.0"

// quizBm25Weights ranks quizzes by title then description (2-column FTS, no
// body). Separate from bm25Weights because quizzes_fts has one fewer column.
const quizBm25Weights = "10.0, 5.0"

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
