package jobs

import (
	"time"
)

func (s *Service) scheduleLoop() {
	defer s.wg.Done()
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-timer.C:
			rows, e := s.DB.Rows(s.ctx, "SELECT key FROM jobs WHERE enabled=1 AND paused=0 AND next_run>0 AND next_run<=?", now.Unix())
			if e != nil {
				continue
			}
			for _, row := range rows {
				_, _ = s.Trigger(row["key"].(string), "scheduled_refresh", "")
			}
		}
	}
}
