package commands

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func resolveProfile(ctx context.Context, db *database.Store, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("profile cannot be empty")
	}

	var id string
	err := db.QueryRowContext(ctx, "SELECT id FROM profiles WHERE id=?", reference).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	rows, err := db.QueryContext(ctx, "SELECT id FROM profiles WHERE display_name=? ORDER BY created_at,id LIMIT 2", reference)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	matches := make([]string, 0, 2)
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			return "", err
		}
		matches = append(matches, id)
	}
	if err = rows.Err(); err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("profile does not exist")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("profile name %q is ambiguous; use the profile ID", reference)
	}
}
