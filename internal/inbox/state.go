package inbox

import (
	"context"
	"fmt"
)

func (s Store) Mark(ctx context.Context, profile string, admin bool, through int64, clear bool) error {
	if through < 0 {
		return fmt.Errorf("invalid notification marker")
	}
	scope, args := visibility(profile, admin)
	var latest int64
	if err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(l.id),0) FROM activity_log l WHERE "+scope, args...).Scan(&latest); err != nil {
		return err
	}
	if through > latest {
		through = latest
	}
	field := "seen_id"
	if clear {
		field = "cleared_id"
	}
	_, err := s.DB.ExecContext(ctx, "INSERT INTO inbox_state(profile_id,"+field+") VALUES(?,?) ON CONFLICT(profile_id) DO UPDATE SET "+field+"=MAX("+field+",excluded."+field+")", profile, through)
	return err
}

func (s Store) Dismiss(ctx context.Context, profile string, admin bool, id int64) error {
	scope, args := visibility(profile, admin)
	result, err := s.DB.ExecContext(ctx, "INSERT INTO inbox_dismissals(profile_id,activity_id) SELECT ?,l.id FROM activity_log l WHERE l.id=? AND "+scope+" ON CONFLICT DO NOTHING", append([]any{profile, id}, args...)...)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}
