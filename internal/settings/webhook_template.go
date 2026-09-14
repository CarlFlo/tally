package settings

import (
	"fmt"
	"strings"
)

// Templates expand decoded string values only; keys are never templated.
func validateTemplate(value any) error {
	switch v := value.(type) {
	case string:
		for _, token := range []string{"event", "key", "level", "message", "show", "time"} {
			v = strings.ReplaceAll(v, "{{"+token+"}}", "")
		}
		if strings.Contains(v, "{{") {
			return fmt.Errorf("unknown payload placeholder")
		}
	case map[string]any:
		for key, item := range v {
			if strings.Contains(key, "{{") {
				return fmt.Errorf("payload placeholders must be in string values, not keys")
			}
			if err := validateTemplate(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			if err := validateTemplate(item); err != nil {
				return err
			}
		}
	}
	return nil
}
