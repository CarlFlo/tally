package backup

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func validateSnapshot(ctx context.Context, dir string, m Manifest, seen map[string]bool) error {
	db, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		return e
	}
	defer db.Close()
	if e = database.Validate(ctx, db); e != nil {
		return e
	}
	if e = database.ValidateSchema(ctx, db); e != nil {
		return e
	}
	var version int
	if e = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); e != nil || version != m.Schema {
		return fmt.Errorf("database schema does not match manifest")
	}
	var id string
	if e = db.QueryRowContext(ctx, "SELECT id FROM profiles WHERE id='user0'").Scan(&id); e != nil {
		return fmt.Errorf("backup has no user0 profile")
	}
	avatars, e := db.QueryContext(ctx, "SELECT avatar FROM profiles WHERE avatar LIKE '%.png'")
	if e != nil {
		return e
	}
	defer avatars.Close()
	for avatars.Next() {
		var name string
		if e = avatars.Scan(&name); e != nil {
			return e
		}
		if !seen["avatars/"+name] {
			return fmt.Errorf("backup is missing a profile avatar")
		}
	}
	if e = avatars.Err(); e != nil {
		return e
	}
	return nil
}
