package database

import (
	"context"
	"fmt"
)

func Validate(ctx context.Context, db Querier) error {
	var result string
	if e := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); e != nil {
		return e
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity check failed")
	}
	rows, e := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if e != nil {
		return e
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("SQLite foreign key check failed")
	}
	return rows.Err()
}
