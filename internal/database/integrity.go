package database

import (
	"context"
	"database/sql"
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
		var table, parent string
		var rowID sql.NullInt64
		var foreignKeyID int
		if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
			return fmt.Errorf("SQLite foreign key check failed: %w", err)
		}
		if rowID.Valid {
			return fmt.Errorf("SQLite foreign key check failed: table %s row %d references %s (constraint %d)", table, rowID.Int64, parent, foreignKeyID)
		}
		return fmt.Errorf("SQLite foreign key check failed: table %s references %s (constraint %d)", table, parent, foreignKeyID)
	}
	return rows.Err()
}
