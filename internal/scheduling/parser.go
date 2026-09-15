package scheduling

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func TimezoneName(timezone string) string {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return "UTC"
	}
	return timezone
}

// Parse accepts the application's single cron dialect in UTC.
func Parse(spec string) (cron.Schedule, error) {
	return ParseInTimezone(spec, "UTC")
}

// ParseInTimezone interprets the five-field cron expression in the deployment timezone.
func ParseInTimezone(spec, timezone string) (cron.Schedule, error) {
	canonical, _, err := Canonical(spec)
	if err != nil {
		return nil, err
	}
	timezone = TimezoneName(timezone)
	if _, err = time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("invalid schedule timezone %q: %v", timezone, err)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	parsed, err := parser.Parse("CRON_TZ=" + timezone + " " + canonical)
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
