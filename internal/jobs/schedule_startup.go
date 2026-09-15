package jobs

import "time"

// refreshScheduleTimes reapplies the deployment timezone to persisted schedules.
func (s *Service) refreshScheduleTimes() error {
	rows, err := s.DB.QueryContext(s.ctx, "SELECT key,schedule,enabled,paused FROM jobs")
	if err != nil {
		return err
	}
	type savedSchedule struct {
		key, spec       string
		enabled, paused bool
	}
	var schedules []savedSchedule
	for rows.Next() {
		var saved savedSchedule
		if err = rows.Scan(&saved.key, &saved.spec, &saved.enabled, &saved.paused); err != nil {
			rows.Close()
			return err
		}
		schedules = append(schedules, saved)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, saved := range schedules {
		var next int64
		if saved.enabled && !saved.paused {
			nextRun, nextErr := s.nextScheduledRun(saved.spec, time.Now())
			if nextErr != nil {
				return nextErr
			}
			next = nextRun.Unix()
		}
		if _, err = s.DB.ExecContext(s.ctx, "UPDATE jobs SET next_run=? WHERE key=?", next, saved.key); err != nil {
			return err
		}
	}
	return nil
}
