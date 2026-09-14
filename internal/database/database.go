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
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	p := filepath.Join(dir, "app.db")
	info, statErr := os.Stat(p)
	existing := statErr == nil && info.Size() > 0
	file, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = file.Close(); e != nil {
		return nil, e
	}
	if e = os.Chmod(p, 0600); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", filepath.ToSlash(p))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db, p}
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, e = db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); e != nil {
		return fail(e)
	}
	if e = s.migrate(ctx, dir, existing); e != nil {
		return fail(e)
	}
	if e = Validate(ctx, db); e != nil {
		return fail(e)
	}
	if e = ValidateSchema(ctx, db); e != nil {
		return fail(e)
	}
	_, e = db.ExecContext(ctx, "UPDATE job_runs SET status='interrupted',ended_at=?,error='Application stopped during this run' WHERE status='running'", time.Now().Unix())
	if e != nil {
		return fail(e)
	}
	return s, nil
}
