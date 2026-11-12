package storage

import (
	"strings"
	"unicode/utf8"
)

func keywordClause(keyword string) (string, bool) {
	tokens := strings.Fields(keyword)
	if len(tokens) == 0 {
		return "", false
	}
	for _, token := range tokens {
		if utf8.RuneCountInString(token) < 3 {
			return "", false
		}
	}
	escaped := make([]string, 0, len(tokens))
	for _, token := range tokens {
		escaped = append(escaped, `"`+strings.ReplaceAll(token, `"`, `""`)+`"*`)
	}
	return strings.Join(escaped, " "), true
}

func escapeLike(keyword string) string {
	var out strings.Builder
	for _, r := range keyword {
		if r == '\\' || r == '%' || r == '_' {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}
