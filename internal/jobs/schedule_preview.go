package jobs

import (
	"time"

	descriptor "github.com/lnquy/cron"

	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

type SchedulePreview struct {
	Expression  string  `json:"expression"`
	Description string  `json:"description"`
	NextRuns    []int64 `json:"next_runs"`
	Timezone    string  `json:"timezone"`
}

// PreviewSchedule describes the exact schedule parsed by the scheduler.
func PreviewSchedule(spec string, now time.Time) (SchedulePreview, error) {
	parsed, err := scheduling.Parse(spec)
	if err != nil {
		return SchedulePreview{}, err
	}
	d, err := descriptor.NewDescriptor(descriptor.Use24HourTimeFormat(true))
	if err != nil {
		return SchedulePreview{}, err
	}
	canonical, fields, err := scheduling.Canonical(spec)
	if err != nil {
		return SchedulePreview{}, err
	}
	description, err := d.ToDescription(canonical, descriptor.Locale_en)
	if err != nil {
		return SchedulePreview{}, err
	}
	if fields[2] != "*" && fields[2] != "?" && fields[4] != "*" && fields[4] != "?" {
		description += "; runs when either the day-of-month or weekday matches"
	}
	preview := SchedulePreview{Expression: spec, Description: description, Timezone: "UTC"}
	for range 3 {
		now = parsed.Next(now.UTC())
		if now.IsZero() {
			break
		}
		preview.NextRuns = append(preview.NextRuns, now.Unix())
	}
	if len(preview.NextRuns) == 0 {
		return SchedulePreview{}, errNoNextRun{}
	}
	return preview, nil
}

type errNoNextRun struct{}

func (errNoNextRun) Error() string { return "schedule has no next occurrence" }
