package commands

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/CarlFlo/tally/internal/database"
)

type profileSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Authentication string `json:"authentication"`
}

func listProfiles(ctx context.Context, db *database.Store) ([]profileSummary, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT p.id,p.display_name,r.is_admin,p.auth_method
		FROM profiles p
		JOIN profile_roles r ON r.profile_id=p.id
		ORDER BY lower(p.display_name),p.created_at,p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []profileSummary
	for rows.Next() {
		var p profileSummary
		var admin int
		var authMethod string
		if err = rows.Scan(&p.ID, &p.Name, &admin, &authMethod); err != nil {
			return nil, err
		}
		if admin == 1 {
			p.Role = "Administrator"
		} else {
			p.Role = "User"
		}
		if authMethod == "password" {
			p.Authentication = "Password"
		} else {
			p.Authentication = "None"
		}
		profiles = append(profiles, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

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
		return "", fmt.Errorf("profile does not exist; run \"tally reset-password\" to list profiles")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("profile name %q is ambiguous; run \"tally reset-password\" and use the profile ID", reference)
	}
}
