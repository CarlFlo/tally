package metadata

import (
	"context"
	"database/sql"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func externalID(ctx context.Context, tx *sql.Tx, kind, external string) (string, error) {
	var id string
	e := tx.QueryRowContext(ctx, "SELECT internal_id FROM external_ids WHERE provider='tvmaze' AND kind=? AND external_id=?", kind, external).Scan(&id)
	if e == nil {
		return id, nil
	}
	if e != sql.ErrNoRows {
		return "", e
	}
	id = database.ID()
	_, e = tx.ExecContext(ctx, "INSERT INTO external_ids VALUES('tvmaze',?,?,?)", kind, external, id)
	return id, e
}
