package metadata

import (
	"strings"

	"golang.org/x/net/html"
)

func Plain(raw string) string {
	var b strings.Builder
	t := html.NewTokenizer(strings.NewReader(raw))
	skip := false
	for {
		switch t.Next() {
		case html.ErrorToken:
			return strings.Join(strings.Fields(b.String()), " ")
		case html.StartTagToken, html.EndTagToken:
			token := t.Token()
			if token.Data == "script" || token.Data == "style" {
				skip = token.Type == html.StartTagToken
			}
			b.WriteByte(' ')
		case html.TextToken:
			if !skip {
				b.Write(t.Text())
			}
		}
	}
}
