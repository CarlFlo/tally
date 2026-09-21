package api

import (
	"strings"
	"testing"
)

func TestNormalizedSearchQueryUsesUnicodeCharactersAndWhitespace(t *testing.T) {
	query, err := normalizedSearchQuery("  Pokémon   世界  ")
	if err != nil {
		t.Fatal(err)
	}
	if query != "Pokémon 世界" {
		t.Fatalf("normalized query = %q", query)
	}

	if _, err = normalizedSearchQuery("界"); err == nil {
		t.Fatal("single Unicode character was accepted")
	}
	if _, err = normalizedSearchQuery(strings.Repeat("界", 201)); err == nil {
		t.Fatal("search over 200 Unicode characters was accepted")
	}
	if _, err = normalizedSearchQuery(string([]byte{0xff, 0xfe})); err == nil {
		t.Fatal("invalid UTF-8 search was accepted")
	}
}
