package backup

import (
	"context"
	"database/sql"
	"fmt"
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
	tables, err := applicationTables(ctx, tx)
	if err != nil {
		return err
	}
	order, err := parentFirstOrder(ctx, tx, tables)
	if err != nil {
		return err
	}

	// Delete every restorable table before inserting any restored rows.
	// SQLite ON DELETE CASCADE actions run immediately even when foreign-key
	// checks are deferred. Interleaving delete+insert by table can therefore
	// erase already-restored child rows when a parent table is deleted later.
	for _, table := range tables {
		if table == "backup_records" {
			continue
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM main."+quoteIdentifier(table)); err != nil {
			return err
		}
	}
	// Keep normal FK enforcement enabled and insert parents before children.
	for _, table := range order {
		name := quoteIdentifier(table)
		if table == "profile_roles" {
			// Inserting profiles fires the first-admin role trigger. Replace those
			// generated rows with the exact role state from the staged backup.
			if _, err = tx.ExecContext(ctx, "DELETE FROM main.profile_roles"); err != nil {
				return err
			}
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

func parentFirstOrder(ctx context.Context, tx *sql.Tx, tables []string) ([]string, error) {
	inRestore := make(map[string]bool, len(tables))
	for _, table := range tables {
		if table != "backup_records" {
			inRestore[table] = true
		}
	}
	dependencies := make(map[string]map[string]bool, len(inRestore))
	for table := range inRestore {
		dependencies[table] = map[string]bool{}
		rows, err := tx.QueryContext(ctx, "PRAGMA main.foreign_key_list("+quoteIdentifier(table)+")")
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, seq int
			var parent, from, to, onUpdate, onDelete, match string
			if err = rows.Scan(&id, &seq, &parent, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				rows.Close()
				return nil, err
			}
			if parent == table {
				rows.Close()
				return nil, fmt.Errorf("self-referencing restore table %s is unsupported", table)
			}
			if inRestore[parent] {
				dependencies[table][parent] = true
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err = rows.Close(); err != nil {
			return nil, err
		}
	}
	done := map[string]bool{}
	order := make([]string, 0, len(inRestore))
	for len(order) < len(inRestore) {
		progress := false
		for _, table := range tables {
			if !inRestore[table] || done[table] {
				continue
			}
			ready := true
			for parent := range dependencies[table] {
				if !done[parent] {
					ready = false
					break
				}
			}
			if ready {
				done[table] = true
				order = append(order, table)
				progress = true
			}
		}
		if !progress {
			return nil, fmt.Errorf("restore schema contains a foreign-key cycle")
		}
	}
	return order, nil
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
