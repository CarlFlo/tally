package jobs

import (
	"context"

	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Service) Start() error {
	if e := (settings.Store{DB: s.DB}).Ensure(s.ctx, s.Config); e != nil {
		return e
	}
	if e := s.initializeSchedules(); e != nil {
		return e
	}
	if _, e := s.DB.ExecContext(s.ctx, "UPDATE show_actions SET status='queued' WHERE status='running'"); e != nil {
		return e
	}
	s.wg.Add(3)
	go s.scheduleLoop()
	go s.queueLoop()
	go s.deliverNotifications()
	return nil
}

func (s *Service) Stop(ctx context.Context) {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	s.cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
