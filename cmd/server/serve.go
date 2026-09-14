package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/api"
	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/backup"
	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/jobs"
	"github.com/CarlFlo/mediaManager/internal/live"
	"github.com/CarlFlo/mediaManager/internal/metadata"
	"github.com/CarlFlo/mediaManager/internal/providers"
	"github.com/CarlFlo/mediaManager/internal/settings"
	"github.com/CarlFlo/mediaManager/internal/torrent"
	"github.com/CarlFlo/mediaManager/web"
)

func serve(ctx context.Context, c config.Config, db *database.Store, b *backup.Service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p, e := providers.New(ctx, db, c.DataDir, c.ProviderConcurrency, c.JobRetries)
	if e != nil {
		return e
	}
	defer p.Close()
	clients := &torrent.ClientStore{DB: db, Control: p}
	m := &metadata.Service{DB: db, Provider: &metadata.TVMaze{Control: p}}
	a := auth.New(db, c)
	c = (settings.Store{DB: db}).LegacyDefaults(ctx, c)
	hub := live.New()
	m.OnChange = hub.Publish
	j := jobs.New(ctx, db, c, m, p, b)
	j.OnChange = hub.Publish
	p.Alert = j.Alert
	if e = j.Start(); e != nil {
		return e
	}
	s := &api.Server{DB: db, Config: c, Auth: a, OIDC: auth.NewOIDC(a, p), Metadata: m, Control: p, Jobs: j, Events: hub, Clients: clients, Assets: web.Assets()}
	server := &http.Server{Addr: c.Addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 32 << 10}
	errors := make(chan error, 1)
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
	j.Stop(shutdown)
	if err := server.Shutdown(shutdown); err != nil {
		server.Close()
	}
	if e != nil && e != http.ErrServerClosed {
		return e
	}
	return nil
}
