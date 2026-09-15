package profiles

import (
	"fmt"
	"regexp"
	"strings"
)

var htmlColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func NormalizeAvatar(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "violet", nil
	}
	switch value {
	case "violet", "mint", "amber", "rose", "blue", "peach":
		return value, nil
	}
	if htmlColor.MatchString(value) {
		return strings.ToUpper(value), nil
	}
	return "", fmt.Errorf("choose a built-in avatar color or a six-digit HTML color such as #4F46E5")
}
