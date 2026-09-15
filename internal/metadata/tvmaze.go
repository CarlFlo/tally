package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/providers"
)

type TVMaze struct {
	Control providers.Requester
	BaseURL string
}

func (m *TVMaze) get(ctx context.Context, path, entity string, ttl time.Duration, out any) error {
	i, _ := ctx.Value(triggerKey{}).(Info)
	if i.Trigger == "" {
		i.Trigger = "user_search"
	}
	base := m.BaseURL
	if base == "" {
		base = "https://api.tvmaze.com"
	}
	r, e := m.Control.Do(ctx, providers.Request{Provider: "tvmaze", URL: base + path, Entity: entity, Trigger: i.Trigger, JobID: i.JobID, TTL: ttl, Force: i.Force})
	if e != nil {
		return e
	}
	if e = json.Unmarshal(r.Body, out); e != nil {
		return fmt.Errorf("TVmaze returned invalid metadata")
	}
	return nil
}

func (m *TVMaze) SearchShows(ctx context.Context, q string) ([]SearchResult, error) {
	q = strings.ToLower(strings.Join(strings.Fields(q), " "))
	out := []SearchResult{}
	e := m.get(ctx, "/search/shows?q="+url.QueryEscape(q), "search", 30*time.Minute, &out)
	for i := range out {
		out[i].Show.Summary = Plain(out[i].Show.Summary)
	}
	return out, e
}

func (m *TVMaze) GetShow(ctx context.Context, id string) (*Show, error) {
	var out Show
	e := m.get(ctx, "/shows/"+url.PathEscape(id), id, 6*time.Hour, &out)
	return &out, e
}

func (m *TVMaze) GetEpisodes(ctx context.Context, id string) ([]Episode, error) {
	out := []Episode{}
	e := m.get(ctx, "/shows/"+url.PathEscape(id)+"/episodes?specials=1", id, 6*time.Hour, &out)
	return out, e
}

func (m *TVMaze) GetSeasons(ctx context.Context, id string) ([]Season, error) {
	out := []Season{}
	e := m.get(ctx, "/shows/"+url.PathEscape(id)+"/seasons", id, 6*time.Hour, &out)
	return out, e
}
