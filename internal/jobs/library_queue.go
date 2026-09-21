package jobs

import (
	"context"
	"log/slog"
	"time"
)

const queueFailureLogInterval = 30 * time.Second

func (s *Service) queueLoop() {
	defer s.wg.Done()
	timer := time.NewTicker(250 * time.Millisecond)
	defer timer.Stop()
	var lastFailureLog time.Time
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
			if s.Metadata == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(s.ctx, s.Config.JobRuntime)
			_, err := s.Metadata.ProcessNext(ctx)
			cancel()
			if err != nil && s.ctx.Err() == nil && (lastFailureLog.IsZero() || time.Since(lastFailureLog) >= queueFailureLogInterval) {
				slog.Warn("metadata queue worker failed", "error", err)
				lastFailureLog = time.Now()
			}
		}
	}
}
