package metadata

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

// Repository persists a complete metadata response in one transaction.
type Repository struct{ DB *database.Store }

func (r Repository) Save(ctx context.Context, external string, show *Show, episodes []Episode, seasons []Season) (string, error) {
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	id, e := externalID(ctx, tx, "show", external)
	if e != nil {
		return "", e
	}
	genres, _ := json.Marshal(show.Genres)
	network := ""
	if show.Network != nil {
		network = show.Network.Name
	} else if show.WebChannel != nil {
		network = show.WebChannel.Name
	}
	now := time.Now()
	next := NextCheck(*show, episodes, now).Add(time.Duration(rand.IntN(1800)) * time.Second)
	_, e = tx.ExecContext(ctx, `INSERT INTO shows(id,name,summary,status,premiered,network,genres,image,rating,runtime,last_checked_at,next_check_at,last_changed_at,provider_updated_at,show_type) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,summary=excluded.summary,status=excluded.status,premiered=excluded.premiered,network=excluded.network,genres=excluded.genres,image=excluded.image,rating=excluded.rating,runtime=excluded.runtime,last_checked_at=excluded.last_checked_at,next_check_at=excluded.next_check_at,last_changed_at=CASE WHEN shows.provider_updated_at<>excluded.provider_updated_at THEN excluded.last_changed_at ELSE shows.last_changed_at END,provider_updated_at=excluded.provider_updated_at,show_type=excluded.show_type`, id, show.Name, Plain(show.Summary), show.Status, show.Premiered, network, string(genres), show.Image.Medium, show.Rating.Average, show.Runtime, now.Unix(), next.Unix(), now.Unix(), show.Updated, show.Type)
	if e != nil {
		return "", e
	}
	for _, mapping := range []struct{ provider, value string }{{"imdb", show.Externals.IMDB}, {"thetvdb", strconv.Itoa(show.Externals.TVDB)}} {
		if mapping.value != "" && mapping.value != "0" {
			_, e = tx.ExecContext(ctx, "INSERT INTO external_ids VALUES(?,'show',?,?) ON CONFLICT(provider,kind,internal_id) DO UPDATE SET external_id=excluded.external_id", mapping.provider, mapping.value, id)
			if e != nil {
				return "", e
			}
		}
	}
	for _, season := range seasons {
		sid, e := externalID(ctx, tx, "season", strconv.Itoa(season.ID))
		if e != nil {
			return "", e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO seasons VALUES(?,?,?,?,?,?,?) ON CONFLICT(show_id,number) DO UPDATE SET name=excluded.name,episode_count=excluded.episode_count,premiere_date=excluded.premiere_date,end_date=excluded.end_date`, sid, id, season.Number, season.Name, season.EpisodeCount, season.PremiereDate, season.EndDate)
		if e != nil {
			return "", e
		}
	}
	for _, ep := range episodes {
		eid, e := externalID(ctx, tx, "episode", strconv.Itoa(ep.ID))
		if e != nil {
			return "", e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO episodes VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET season=excluded.season,number=excluded.number,name=excluded.name,summary=excluded.summary,airdate=excluded.airdate,airstamp=excluded.airstamp,runtime=excluded.runtime,type=excluded.type`, eid, id, ep.Season, ep.Number, ep.Name, Plain(ep.Summary), ep.Airdate, ep.Airstamp, ep.Runtime, ep.Type)
		if e != nil {
			return "", e
		}
	}
	return id, tx.Commit()
}
