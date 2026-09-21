package commands

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/api"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/jobs"
	"github.com/CarlFlo/tally/internal/live"
	"github.com/CarlFlo/tally/internal/localization"
	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/torrent"
	"github.com/CarlFlo/tally/web"
)

const (
	jobShutdownTimeout  = 10 * time.Second
	httpShutdownTimeout = 10 * time.Second
)

func serve(ctx context.Context, c config.Config, db *database.Store, b *backup.Service) (time.Time, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p, err := providers.New(ctx, db, c.DataDir, c.ProviderConcurrency, c.JobRetries)
	if err != nil {
		return time.Time{}, err
	}
	started := false
	providerClosed := false
	defer func() {
		if !providerClosed {
			if closeErr := p.Close(); closeErr != nil && started {
				slog.Error("Shutdown: provider close failed", "error", closeErr)
			}
		}
	}()

	clients := &torrent.ClientStore{DB: db, Control: p}
	m := &metadata.Service{DB: db, Provider: &metadata.TVMaze{Control: p}}
	a := auth.New(db, c)
	hub := live.New()
	archiveWatcher, watchErr := backup.WatchArchives(b.Path, func() { hub.Publish("", "backups") })
	if watchErr != nil {
		slog.Warn("Backup archive watcher unavailable; archive changes require a page reload", "error", watchErr)
	}
	archiveWatcherClosed := false
	defer func() {
		if archiveWatcher != nil && !archiveWatcherClosed {
			if closeErr := archiveWatcher.Close(); closeErr != nil && started {
				slog.Warn("Shutdown: backup archive watcher close failed", "error", closeErr)
			}
		}
	}()
	locales, err := localization.New(c.DataDir, func() { hub.Publish("", "locales") })
	if err != nil {
		return time.Time{}, err
	}
	localesClosed := false
	defer func() {
		if !localesClosed {
			if closeErr := locales.Close(); closeErr != nil && started {
				slog.Warn("Shutdown: localization watcher close failed", "error", closeErr)
			}
		}
	}()
	m.OnChange = hub.Publish
	j := jobs.New(ctx, db, c, m, p, b)
	j.OnChange = hub.Publish
	p.Alert = j.Alert
	operatorControl, err := startOperatorServer(ctx, c, db, b, hub, j)
	if err != nil {
		return time.Time{}, err
	}
	defer operatorControl.Close()
	if err = j.Start(); err != nil {
		return time.Time{}, err
	}

	s := &api.Server{DB: db, Backup: b, Config: c, Auth: a, Metadata: m, Control: p, Jobs: j, Events: hub, Locales: locales, Clients: clients, Assets: web.Assets()}
	server := &http.Server{Addr: c.Addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 32 << 10}
	errors := make(chan error, 1)
	started = true
	go func() {
		slog.Info("Tally is ready", "address", c.Addr, "schema", database.Version)
		errors <- server.ListenAndServe()
	}()

	select {
	case err = <-errors:
		if err != http.ErrServerClosed {
			cancel()
		}
	case <-ctx.Done():
	}

	shutdownStarted := time.Now()
	slog.Info("Shutdown: stopping background services")

	jobsCtx, stopJobs := context.WithTimeout(context.Background(), jobShutdownTimeout)
	j.Stop(jobsCtx)
	if jobsCtx.Err() != nil {
		slog.Warn("Shutdown: background jobs did not stop before timeout", "error", jobsCtx.Err())
	}
	stopJobs()

	if archiveWatcher != nil {
		if closeErr := archiveWatcher.Close(); closeErr != nil {
			slog.Warn("Shutdown: backup archive watcher close failed", "error", closeErr)
		}
		archiveWatcherClosed = true
	}

	if closeErr := locales.Close(); closeErr != nil {
		slog.Warn("Shutdown: localization watcher close failed", "error", closeErr)
	}
	localesClosed = true
	hub.Close()

	httpCtx, stopHTTP := context.WithTimeout(context.Background(), httpShutdownTimeout)
	slog.Info("Shutdown: draining HTTP connections")
	if shutdownErr := server.Shutdown(httpCtx); shutdownErr != nil {
		slog.Warn("Shutdown: HTTP drain did not finish; forcing close", "error", shutdownErr)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("Shutdown: forced HTTP close failed", "error", closeErr)
		}
	}
	stopHTTP()

	if closeErr := p.Close(); closeErr != nil {
		slog.Error("Shutdown: provider close failed", "error", closeErr)
	}
	providerClosed = true

	if err != nil && err != http.ErrServerClosed {
		return shutdownStarted, err
	}
	return shutdownStarted, nil
}
