package backup

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func (s *Service) snapshotFiles(ctx context.Context, snapshot string) (Manifest, map[string]string, error) {
	manifest := Manifest{1, database.Version, time.Now().UTC().Format(time.RFC3339), map[string]string{}}
	files := map[string]string{"app.db": snapshot}
	avatarDB, e := sql.Open("sqlite", filepath.ToSlash(snapshot))
	if e != nil {
		return manifest, nil, e
	}
	if e = avatarDB.QueryRowContext(ctx, "PRAGMA user_version").Scan(&manifest.Schema); e != nil {
		avatarDB.Close()
		return manifest, nil, e
	}
	rows, e := avatarDB.QueryContext(ctx, "SELECT avatar FROM profiles WHERE avatar LIKE '%.png'")
	if e != nil {
		avatarDB.Close()
		return manifest, nil, e
	}
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			break
		}
		if !safeName("avatars/" + name) {
			e = fmt.Errorf("unsafe avatar path in database")
			break
		}
		files["avatars/"+name] = filepath.Join(s.DataDir, "avatars", name)
	}
	rowsErr := rows.Err()
	rows.Close()
	avatarDB.Close()
	if e != nil {
		return manifest, nil, e
	}
	if rowsErr != nil {
		return manifest, nil, rowsErr
	}

	return manifest, files, nil
}
