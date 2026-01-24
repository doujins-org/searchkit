package normalize

import (
	"strings"
	"unicode"
)

func isLetterOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// normalizeIntraTokenHyphens replaces '-' that appear between two letters/numbers
// with a space. This prevents Postgres websearch_to_tsquery from interpreting
// hyphenated tokens like "two-factor" as a NOT query.
func normalizeIntraTokenHyphens(s string) string {
	rs := []rune(s)
	if len(rs) == 0 {
		return s
	}
	for i := 1; i < len(rs)-1; i++ {
		if rs[i] != '-' {
			continue
		}
		if isLetterOrNumber(rs[i-1]) && isLetterOrNumber(rs[i+1]) {
			rs[i] = ' '
		}
	}
	return string(rs)
}

func stripLeadingHyphens(tok string) string {
	for strings.HasPrefix(tok, "-") {
		tok = strings.TrimPrefix(tok, "-")
	}
	return tok
}

// QueryForEmbedding returns a cleaned query string intended for embedding models
// and non-FTS lexical backends.
//
// Behavior:
//   - Hyphenated tokens are normalized (e.g. "two-factor" -> "two factor").
//   - Leading '-' is treated as punctuation (e.g. "-factor" -> "factor").
//   - Natural language tokens like "not" are preserved (semantic models can
//     interpret them as plain language).
func QueryForEmbedding(input string) string {
	q := normalizeIntraTokenHyphens(strings.TrimSpace(input))
	if q == "" {
		return ""
	}
	parts := strings.Fields(q)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = stripLeadingHyphens(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, " ")
}

// QueryForFTS returns a cleaned query string intended for Postgres
// websearch_to_tsquery.
//
// Behavior:
//   - Hyphenated tokens are normalized (e.g. "two-factor" -> "two factor").
//   - Leading '-' is treated as punctuation and removed from user input
//     (e.g. "-factor" behaves like "factor").
//   - Natural language negation "not X" is converted to "-X" for FTS only.
//
// NOTE: This only supports single-token negation (next whitespace-delimited
// token). Quoted phrase negation (e.g. `not "two factor"`) is not handled.
func QueryForFTS(input string) string {
	q := normalizeIntraTokenHyphens(strings.TrimSpace(input))
	if q == "" {
		return ""
	}

	parts := strings.Fields(q)
	out := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		p := stripLeadingHyphens(parts[i])
		if p == "" {
			continue
		}
		if strings.EqualFold(p, "not") && i+1 < len(parts) {
			next := stripLeadingHyphens(parts[i+1])
			i++
			if next == "" {
				continue
			}
			out = append(out, "-"+next)
			continue
		}
		out = append(out, p)
	}

	return strings.Join(out, " ")
}
