package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (o *OIDC) MapIdentity(ctx context.Context, issuer, subject, name string) (string, error) {
	tx, e := o.Auth.DB.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var id string
	e = tx.QueryRowContext(ctx, "SELECT profile_id FROM profile_identities WHERE issuer=? AND subject=?", issuer, subject).Scan(&id)
	if e == nil {
		return id, nil
	}
	if e != sql.ErrNoRows {
		return "", e
	}
	if !o.Auth.Config.OIDCAutoCreate {
		return "", fmt.Errorf("this identity has no profile; ask the operator")
	}
	var count, identities int
	_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles").Scan(&count)
	_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM profile_identities").Scan(&identities)
	// First identity claims user0 only on a pristine deployment, preserving existing profile ownership.
	var used int
	_ = tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM profile_shows WHERE profile_id='user0')+(SELECT COUNT(*) FROM local_credentials)").Scan(&used)
	if count == 1 && identities == 0 && used == 0 {
		id = "user0"
	} else {
		if count >= o.Auth.Config.MaxProfiles {
			return "", fmt.Errorf("profile limit reached")
		}
		var n int
		if e = tx.QueryRowContext(ctx, "UPDATE counters SET value=value+1 WHERE key='profile' RETURNING value").Scan(&n); e != nil {
			return "", e
		}
		id = fmt.Sprintf("user%d", n)
		if strings.TrimSpace(name) == "" {
			name = "New profile"
		}
		if len(name) > 80 {
			name = name[:80]
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO profiles VALUES(?,?,'violet',?)", id, name, time.Now().Unix()); e != nil {
			return "", e
		}
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO profile_identities VALUES(?,?,?)", issuer, subject, id); e != nil {
		return "", e
	}
	return id, tx.Commit()
}
