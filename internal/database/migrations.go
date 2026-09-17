package database

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

//go:embed schema.sql
var schema string

//go:embed migration_002.sql
var migration002 string

//go:embed migration_003.sql
var migration003 string

//go:embed migration_004.sql
var migration004 string

//go:embed migration_005.sql
var migration005 string

//go:embed migration_006.sql
var migration006 string

//go:embed migration_007.sql
var migration007 string

//go:embed migration_008.sql
var migration008 string

const Version = 8

func (s *Store) migrate(ctx context.Context, dir string, existing bool) error {
	var e error
	var version int
	if e = s.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version > Version {
		return fmt.Errorf("database schema %d is newer than supported %d; downgrade refused", version, Version)
	}
	if version < Version {
		if existing {
			backup := filepath.Join(dir, fmt.Sprintf("pre-upgrade-v%d-%d.db", version, time.Now().UnixNano()))
			if e = s.Snapshot(ctx, backup); e != nil {
				return fmt.Errorf("pre-upgrade backup failed: %w", e)
			}
		}
		tx, e := s.BeginTx(ctx, nil)
		if e != nil {
			return e
		}
		for next := version + 1; next <= Version && e == nil; next++ {
			switch next {
			case 1:
				_, e = tx.ExecContext(ctx, schema)
			case 2:
				_, e = tx.ExecContext(ctx, migration002)
			case 3:
				_, e = tx.ExecContext(ctx, migration003)
			case 4:
				_, e = tx.ExecContext(ctx, migration004)
			case 5:
				_, e = tx.ExecContext(ctx, migration005)
			case 6:
				_, e = tx.ExecContext(ctx, migration006)
			case 7:
				_, e = tx.ExecContext(ctx, migration007)
			case 8:
				_, e = tx.ExecContext(ctx, migration008)
			default:
				e = fmt.Errorf("no migration to schema %d", next)
			}
		}
		if e == nil {
			_, e = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", Version))
		}
		if e == nil {
			e = Validate(ctx, tx)
		}
		if e == nil {
			e = ValidateSchema(ctx, tx)
		}
		if e == nil {
			e = tx.Commit()
		} else {
			tx.Rollback()
		}
		if e != nil {
			_ = os.WriteFile(filepath.Join(dir, "migration-failed.json"), []byte(`{"migration_failed":true}`), 0600)
			return fmt.Errorf("migration failed: %w", e)
		}
	}
	return nil
}
