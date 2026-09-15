package jobs

import (
	"time"

	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

func (s *Service) scheduleTimezone() string {
	return scheduling.TimezoneName(s.Config.Timezone)
}

func (s *Service) nextScheduledRun(spec string, now time.Time) (time.Time, error) {
	parsed, err := scheduling.ParseInTimezone(spec, s.scheduleTimezone())
	if err != nil {
		return time.Time{}, err
	}
	return parsed.Next(now), nil
}
