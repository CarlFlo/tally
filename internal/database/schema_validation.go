package database

import (
	"context"
	"database/sql"
	"fmt"
)

type Querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func ValidateSchema(ctx context.Context, db Querier) error {
	var version int
	if e := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version < 1 || version > Version {
		return fmt.Errorf("unsupported database schema %d", version)
	}
	if version >= 2 {
		rows, e := db.QueryContext(ctx, "SELECT id,adapter,fields,revision,updated_at FROM download_client_settings LIMIT 0")
		if e != nil {
			return fmt.Errorf("database client settings schema is incomplete: %w", e)
		}
		rows.Close()
		var count int
		if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM download_client_settings WHERE id=1").Scan(&count); e != nil {
			return e
		}
		if count != 1 {
			return fmt.Errorf("database client settings row is missing")
		}
	}
	if version >= 3 {
		for _, query := range []string{
			"SELECT key,data,revision FROM application_settings LIMIT 0",
			"SELECT profile_id,show_id,favorite FROM profile_shows LIMIT 0",
			"SELECT enabled,paused,failures,revision FROM jobs LIMIT 0",
			"SELECT profile_id,external_id,name,desired,status,revision,error,show_id,updated_at FROM show_actions LIMIT 0",
		} {
			rows, e := db.QueryContext(ctx, query)
			if e != nil {
				return fmt.Errorf("database settings/queue schema is incomplete: %w", e)
			}
			rows.Close()
		}
	}
	if version >= 5 {
		for _, query := range []string{"SELECT profile_id,seen_id,cleared_id FROM inbox_state LIMIT 0", "SELECT profile_id,activity_id FROM inbox_dismissals LIMIT 0"} {
			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return fmt.Errorf("database inbox schema is incomplete: %w", err)
			}
			rows.Close()
		}
	}
	if version >= 4 {
		for _, query := range []string{
			"SELECT id,action,profile_id,show_id,show_name,message,created_at FROM activity_log LIMIT 0",
			"SELECT id,theme,updated_at FROM browser_preferences LIMIT 0",
			"SELECT key,value FROM notification_state LIMIT 0",
			"SELECT id,event_key,event,message,show_name,occurred_at,available_at,source_key,level,status FROM notification_outbox LIMIT 0",
		} {
			rows, e := db.QueryContext(ctx, query)
			if e != nil {
				return fmt.Errorf("database activity/notification schema is incomplete: %w", e)
			}
			rows.Close()
		}
	}
	for _, query := range []string{
		"SELECT id,display_name,avatar,created_at FROM profiles LIMIT 0",
		"SELECT key,value FROM counters LIMIT 0",
		"SELECT profile_id,data FROM profile_preferences LIMIT 0",
		"SELECT issuer,subject,profile_id FROM profile_identities LIMIT 0",
		"SELECT profile_id,hash,must_change FROM local_credentials LIMIT 0",
		"SELECT id,profile_id,last_seen,expires_at,restricted FROM sessions LIMIT 0",
		"SELECT id,name,next_check_at,provider_updated_at FROM shows LIMIT 0",
		"SELECT id,show_id,number FROM seasons LIMIT 0",
		"SELECT id,show_id,season,number,airdate,airstamp FROM episodes LIMIT 0",
		"SELECT provider,kind,external_id,internal_id FROM external_ids LIMIT 0",
		"SELECT profile_id,show_id FROM profile_shows LIMIT 0",
		"SELECT profile_id,episode_id,watched,downloaded FROM profile_episode_state LIMIT 0",
		"SELECT key,type,schedule,next_run FROM jobs LIMIT 0",
		"SELECT id,job_key,status,attempt FROM job_runs LIMIT 0",
		"SELECT provider,state,blocked_until FROM provider_state LIMIT 0",
		"SELECT id,provider,reason FROM provider_requests LIMIT 0",
		"SELECT day,provider,requests FROM provider_request_aggregates LIMIT 0",
		"SELECT id,key,active FROM alerts LIMIT 0",
		"SELECT id,filename,kind FROM backup_records LIMIT 0",
		"SELECT id,profile_id,query FROM torrent_search_history LIMIT 0",
		"SELECT id,profile_id,idempotency_key,request_hash,status FROM torrent_send_history LIMIT 0",
	} {
		rows, e := db.QueryContext(ctx, query)
		if e != nil {
			return fmt.Errorf("database schema is incomplete: %w", e)
		}
		rows.Close()
	}
	return nil
}
