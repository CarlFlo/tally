package settings

import "fmt"

type Backups struct {
	Keep int `json:"keep"`
}

func ValidateBackups(value Backups) error {
	if value.Keep < 1 || value.Keep > 1000 {
		return fmt.Errorf("keep between 1 and 1000 automatic backups")
	}
	return nil
}
