package jobs

import (
	"time"

	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

func (s *Service) initializeSchedules() error {
	for _, entry := range []struct {
		key, spec, fallback string
		enabled             bool
	}{
		{"metadata", "0 * * * *", "0 * * * *", true},
		{"maintenance", "30 3 * * *", "30 3 * * *", true},
		{"backup", "0 3 * * *", "0 3 * * *", true},
	} {
		spec := entry.spec
		if spec == "" {
			spec = entry.fallback
		}
		parsed, e := scheduling.Parse(spec)
		if e != nil {
			return e
		}
		var next int64
		if entry.enabled {
			next = parsed.Next(time.Now().UTC()).Unix()
		}
		if _, e = s.DB.ExecContext(s.ctx, "INSERT INTO jobs(key,type,schedule,next_run,enabled) VALUES(?,?,?,?,?) ON CONFLICT DO NOTHING", entry.key, entry.key, spec, next, entry.enabled); e != nil {
			return e
		}
	}
	return s.refreshScheduleTimes()
}
