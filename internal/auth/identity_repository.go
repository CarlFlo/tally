package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/database"
)

func (o *OIDC) MapIdentity(ctx context.Context, issuer, subject, name string) (string, error) {
	tx, err := o.Auth.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx, "SELECT profile_id FROM profile_identities WHERE issuer=? AND subject=?", issuer, subject).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	if !o.Auth.Config.OIDCAutoCreate {
		return "", fmt.Errorf("this identity has no profile; ask an administrator")
	}

	var count, identities int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles").Scan(&count); err != nil {
		return "", err
	}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM profile_identities").Scan(&identities); err != nil {
		return "", err
	}

	// Preserve ownership on an unused single-profile installation without
	// coupling that profile to a special ID or permanent role.
	if count == 1 && identities == 0 {
		var used int
		if err = tx.QueryRowContext(ctx, `SELECT p.id,
			(SELECT COUNT(*) FROM profile_shows WHERE profile_id=p.id)+
			(SELECT COUNT(*) FROM local_credentials WHERE profile_id=p.id)
			FROM profiles p LIMIT 1`).Scan(&id, &used); err != nil {
			return "", err
		}
		if used != 0 {
			id = ""
		}
	}

	if id == "" {
		if count >= o.Auth.Config.MaxProfiles {
			return "", fmt.Errorf("profile limit reached")
		}
		id = database.ID()
		name = strings.TrimSpace(name)
		if name == "" {
			name = "New profile"
		}
		if len(name) > 80 {
			name = name[:80]
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO profiles(id,display_name,avatar,created_at) VALUES(?,?,'violet',?)", id, name, time.Now().Unix()); err != nil {
			return "", err
		}
		if err = activity.Record(ctx, tx, activity.Event{Action: "profile_created", Profile: id, Message: "Created profile " + name}); err != nil {
			return "", err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO profile_identities VALUES(?,?,?)", issuer, subject, id); err != nil {
		return "", err
	}
	return id, tx.Commit()
}
