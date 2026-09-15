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
	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/torrent"
	"github.com/CarlFlo/tally/web"
)

func serve(ctx context.Context, c config.Config, db *database.Store, b *backup.Service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p, e := providers.New(ctx, db, c.DataDir, c.ProviderConcurrency, c.JobRetries)
	if e != nil {
		return e
	}
	started := false
	defer func() {
		if started {
			slog.Info("Shutdown: closing provider connections and cache")
		}
		if err := p.Close(); err != nil && started {
			slog.Error("Shutdown: provider close failed", "error", err)
		}
	}()

	clients := &torrent.ClientStore{DB: db, Control: p}
	m := &metadata.Service{DB: db, Provider: &metadata.TVMaze{Control: p}}
	a := auth.New(db, c)
	hub := live.New()
	m.OnChange = hub.Publish
	j := jobs.New(ctx, db, c, m, p, b)
	j.OnChange = hub.Publish
	p.Alert = j.Alert
	if e = j.Start(); e != nil {
		return e
	}

	s := &api.Server{DB: db, Backup: b, Config: c, Auth: a, OIDC: auth.NewOIDC(a, p), Metadata: m, Control: p, Jobs: j, Events: hub, Clients: clients, Assets: web.Assets()}
	server := &http.Server{Addr: c.Addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 32 << 10}
	errors := make(chan error, 1)
	started = true
	go func() {
		slog.Info("Tally is ready", "address", c.Addr, "auth_mode", c.AuthMode, "schema", database.Version)
		errors <- server.ListenAndServe()
	}()

	select {
	case e = <-errors:
		if e != http.ErrServerClosed {
			cancel()
		}
	case <-ctx.Done():
	}

	shutdown, done := context.WithTimeout(context.Background(), 25*time.Second)
	defer done()

	slog.Info("Shutdown: stopping background jobs")
	j.Stop(shutdown)

	slog.Info("Shutdown: closing live connections")
	hub.Close()

	slog.Info("Shutdown: draining HTTP connections")
	if err := server.Shutdown(shutdown); err != nil {
		slog.Warn("Shutdown: HTTP drain did not finish; forcing close", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("Shutdown: forced HTTP close failed", "error", closeErr)
		}
	}

	if e != nil && e != http.ErrServerClosed {
		return e
	}
	return nil
}
