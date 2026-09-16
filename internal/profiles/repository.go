package profiles

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/database"
)

var (
	ErrLimit     = errors.New("maximum number of profiles reached")
	ErrNotFound  = errors.New("profile not found")
	ErrLastAdmin = errors.New("at least one administrator is required while profiles remain")
)

type Profile struct {
	ID     string `json:"id"`
	Name   string `json:"display_name"`
	Avatar string `json:"avatar"`
	Locale string `json:"locale"`
	Admin  bool   `json:"is_admin"`
}

type Repository struct {
	DB    *database.Store
	Limit int
}

func (r Repository) Create(ctx context.Context, name, avatar, locale, hash, actor string) (Profile, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	var err error
	avatar, err = NormalizeAvatar(avatar)
	if err != nil {
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
	if locale == "" {
		locale = "en"
	}
	profile := Profile{ID: database.ID(), Name: name, Avatar: avatar, Locale: locale}
	if _, err = tx.ExecContext(ctx, "INSERT INTO profiles(id,display_name,avatar,created_at,locale) VALUES(?,?,?,?,?)", profile.ID, name, avatar, time.Now().Unix(), locale); err != nil {
		return Profile{}, err
	}
	if hash != "" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO local_credentials VALUES(?,?,0)", profile.ID, hash); err != nil {
			return Profile{}, err
		}
	}
	if err = tx.QueryRowContext(ctx, "SELECT is_admin FROM profile_roles WHERE profile_id=?", profile.ID).Scan(&profile.Admin); err != nil {
		return Profile{}, err
	}
	if actor == "" {
		actor = profile.ID
	}
	actorName := name
	if actor != profile.ID {
		actorName = profileDisplayName(ctx, tx, actor)
	}
	if err = activity.Record(ctx, tx, activity.Event{Action: "profile_created", Profile: actor, Message: actorName + " created profile " + name}); err != nil {
		return Profile{}, err
	}
	return profile, tx.Commit()
}
