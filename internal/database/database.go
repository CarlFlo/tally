package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	*sql.DB
	Path string
}

func Open(ctx context.Context, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "app.db")
	info, statErr := os.Stat(p)
	existing := statErr == nil && info.Size() > 0
	file, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	if err = os.Chmod(p, 0600); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.ToSlash(p))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db, p}
	fail := func(err error) (*Store, error) { db.Close(); return nil, err }
	if _, err = db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000; PRAGMA wal_autocheckpoint=1000;"); err != nil {
		return fail(err)
	}
	if err = s.migrate(ctx, dir, existing); err != nil {
		return fail(err)
	}
	if err = Validate(ctx, db); err != nil {
		return fail(err)
	}
	if err = ValidateSchema(ctx, db); err != nil {
		return fail(err)
	}
	_, err = db.ExecContext(ctx, "UPDATE job_runs SET status='interrupted',ended_at=?,error='Application stopped during this run' WHERE status='running'", time.Now().Unix())
	if err != nil {
		return fail(err)
	}
	return s, nil
}

func (s *Store) Checkpoint(ctx context.Context, truncate bool) error {
	mode := "PASSIVE"
	if truncate {
		mode = "TRUNCATE"
	}
	_, err := s.ExecContext(ctx, "PRAGMA wal_checkpoint("+mode+")")
	return err
}
