// Package notifications delivers subscribed events through the provider coordinator.
package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

type Service struct {
	DB        *database.Store
	Requester providers.Requester
	Timezone  string
	OnChange  func(string, ...string)
}

func (s *Service) changedProfile(profile string, resources ...string) {
	if s.OnChange != nil {
		s.OnChange(profile, resources...)
	}
}

func (s *Service) Run(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-wake:
		}
		call, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.Tick(call, time.Now()); err != nil && ctx.Err() == nil {
			slog.Warn("notification worker failed", "error", err)
		}
		cancel()
	}
}

func (s *Service) Tick(ctx context.Context, now time.Time) error {
	var config settings.Webhook
	if _, err := (settings.Store{DB: s.DB}).Load(ctx, "notifications", &config); err != nil {
		return err
	}
	config = config.Defaults(s.Timezone)
	if err := s.collectBellReleases(ctx, now); err != nil {
		return err
	}
	if err := s.collectActivity(ctx, config); err != nil {
		return err
	}
	if err := s.collectReleases(ctx, config, now); err != nil {
		return err
	}
	return s.deliver(ctx, now)
}
