package auth

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func (s *Service) Policy(password string) error {
	n := utf8.RuneCountInString(password)
	if n < s.Config.PasswordMin || n > s.Config.PasswordMax {
		return fmt.Errorf("password must contain %d–%d characters", s.Config.PasswordMin, s.Config.PasswordMax)
	}
	if !utf8.ValidString(password) {
		return fmt.Errorf("password contains invalid characters")
	}
	for _, r := range password {
		if unicode.IsControl(r) {
			return fmt.Errorf("password cannot contain control characters")
		}
	}

	return nil
}
