package scheduling

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
)

// Parse accepts the application's single cron dialect and always schedules in UTC.
func Parse(spec string) (cron.Schedule, error) {
	canonical, _, err := Canonical(spec)
	if err != nil {
		return nil, err
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	parsed, err := parser.Parse("CRON_TZ=UTC " + canonical)
	if err != nil {
		return nil, fmt.Errorf("invalid five-field cron schedule: %v", err)
	}
	return parsed, nil
}

// Canonical normalizes accepted compatibility forms without changing stored input.
func Canonical(spec string) (string, []string, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return "", nil, fmt.Errorf("enter a valid five-field cron schedule")
	}
	if fields[0] == "*/60" {
		fields[0] = "0"
	}
	return strings.Join(fields, " "), fields, nil
}
