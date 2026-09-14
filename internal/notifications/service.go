// Package notifications delivers subscribed events through the provider coordinator.
package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/providers"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

type Service struct {
	DB        *database.Store
	Requester providers.Requester
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
	config = config.Defaults()
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
