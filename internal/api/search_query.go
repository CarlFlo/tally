package api

import (
	"strings"
	"unicode/utf8"
)

func normalizedSearchQuery(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", bad("search contains invalid characters")
	}
	query := strings.Join(strings.Fields(raw), " ")
	length := utf8.RuneCountInString(query)
	if length < 2 || length > 200 {
		return "", bad("search with 2–200 characters")
	}
	return query, nil
}
