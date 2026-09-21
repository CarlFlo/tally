package profiles

import (
	"context"
	"database/sql"
	"errors"
)

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func profileDisplayName(ctx context.Context, db rowQuerier, id string) (string, error) {
	if id == "" {
		return "System", nil
	}
	var name string
	err := db.QueryRowContext(ctx, "SELECT display_name FROM profiles WHERE id=?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) || name == "" {
		return "Deleted profile", nil
	}
	if err != nil {
		return "", err
	}
	return name, nil
}
