package profiles

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/CarlFlo/tally/internal/activity"
)

func roleError(err error) error {
	if err != nil && strings.Contains(err.Error(), ErrLastAdmin.Error()) {
		return ErrLastAdmin
	}
	return err
}

func (r Repository) SetAdmin(ctx context.Context, actor, target string, admin bool) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var name string
	var current bool
	if err = tx.QueryRowContext(ctx, `SELECT p.display_name,r.is_admin
		FROM profiles p JOIN profile_roles r ON r.profile_id=p.id WHERE p.id=?`, target).Scan(&name, &current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if current == admin {
		return nil
	}
	if _, err = tx.ExecContext(ctx, "UPDATE profile_roles SET is_admin=? WHERE profile_id=?", admin, target); err != nil {
		return roleError(err)
	}
	action, verb := "admin_revoked", " removed administrator access from "
	if admin {
		action, verb = "admin_granted", " granted administrator access to "
	}
	actorName, err := profileDisplayName(ctx, tx, actor)
	if err != nil {
		return err
	}
	if err = activity.Record(ctx, tx, activity.Event{Action: action, Profile: actor, Message: actorName + verb + name}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r Repository) Delete(ctx context.Context, actor, target string) (string, bool, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()

	var name string
	var admin bool
	if err = tx.QueryRowContext(ctx, `SELECT p.display_name,r.is_admin
		FROM profiles p JOIN profile_roles r ON r.profile_id=p.id WHERE p.id=?`, target).Scan(&name, &admin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, ErrNotFound
		}
		return "", false, err
	}
	actorName := name
	if actor != target {
		actorName, err = profileDisplayName(ctx, tx, actor)
		if err != nil {
			return "", false, err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM profiles WHERE id=?", target); err != nil {
		return "", false, roleError(err)
	}
	if err = activity.Record(ctx, tx, activity.Event{Action: "profile_deleted", Profile: actor, Message: actorName + " deleted profile " + name}); err != nil {
		return "", false, err
	}
	if err = tx.Commit(); err != nil {
		return "", false, roleError(err)
	}
	return name, admin, nil
}
