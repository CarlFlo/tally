package jobs

import (
	"context"
	"time"
)

func (s *Service) queueLoop() {
	defer s.wg.Done()
	timer := time.NewTicker(250 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
			if s.Metadata == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(s.ctx, s.Config.JobRuntime)
			_, _ = s.Metadata.ProcessNext(ctx)
			cancel()
		}
	}
}
