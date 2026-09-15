package profiles

import (
	"context"
	"database/sql"
)

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func profileDisplayName(ctx context.Context, db rowQuerier, id string) string {
	if id == "" {
		return "System"
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT display_name FROM profiles WHERE id=?", id).Scan(&name); err != nil || name == "" {
		return "Deleted profile"
	}
	return name
}
