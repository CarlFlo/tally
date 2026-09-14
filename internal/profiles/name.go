package profiles

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ValidateName(name string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(name))
	if n < 1 || n > 80 || !utf8.ValidString(name) {
		return fmt.Errorf("display name must contain 1-80 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("display name cannot contain control characters")
		}
	}
	return nil
}
