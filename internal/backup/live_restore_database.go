package backup

import (
	"context"
	"database/sql"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func replaceDatabaseState(ctx context.Context, db *database.Store, stagedPath string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "ATTACH DATABASE ? AS restore", stagedPath); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "DETACH DATABASE restore")
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "PRAGMA defer_foreign_keys=ON"); err != nil {
		return err
	}
	tables, err := applicationTables(ctx, tx)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if table == "backup_records" {
			continue
		}
		name := quoteIdentifier(table)
		if _, err = tx.ExecContext(ctx, "DELETE FROM main."+name); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO main."+name+" SELECT * FROM restore."+name); err != nil {
			return err
		}
	}
	if err = restoreSequences(ctx, tx); err != nil {
		return err
	}
	if err = database.Validate(ctx, tx); err != nil {
		return err
	}
	if err = database.ValidateSchema(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func applicationTables(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT name FROM main.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func restoreSequences(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM main.sqlite_sequence"); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO main.sqlite_sequence(name,seq) SELECT name,seq FROM restore.sqlite_sequence")
	return err
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
