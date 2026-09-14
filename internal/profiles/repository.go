package profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/database"
)

var ErrLimit = errors.New("maximum number of profiles reached")

type Profile struct {
	ID     string `json:"id"`
	Name   string `json:"display_name"`
	Avatar string `json:"avatar"`
}
type Repository struct {
	DB    *database.Store
	Limit int
}

func (r Repository) Create(ctx context.Context, name, avatar, hash, actor string) (Profile, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles").Scan(&count); err != nil {
		return Profile{}, err
	}
	if count >= r.Limit {
		return Profile{}, ErrLimit
	}
	var n int
	if err = tx.QueryRowContext(ctx, "UPDATE counters SET value=value+1 WHERE key='profile' RETURNING value").Scan(&n); err != nil {
		return Profile{}, err
	}
	profile := Profile{ID: fmt.Sprintf("user%d", n), Name: name, Avatar: avatar}
	if _, err = tx.ExecContext(ctx, "INSERT INTO profiles VALUES(?,?,?,?)", profile.ID, name, avatar, time.Now().Unix()); err != nil {
		return Profile{}, err
	}
	if hash != "" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO local_credentials VALUES(?,?,0)", profile.ID, hash); err != nil {
			return Profile{}, err
		}
	}
	if actor == "" {
		actor = profile.ID
	}
	if err = activity.Record(ctx, tx, activity.Event{Action: "profile_created", Profile: actor, Message: "Created profile " + name}); err != nil {
		return Profile{}, err
	}
	return profile, tx.Commit()
}
