package metadata

import (
	"context"
	"sort"
	"time"
)

// Suggestions are rated shows from recent US broadcast and worldwide streaming
// schedules, not a claim of a global popularity chart. Six small cached queries.
func (m *TVMaze) SuggestShows(ctx context.Context) ([]SearchResult, error) {
	unique := map[int]Show{}
	var lastErr error
	var successes int
	for _, offset := range []int{0, -1, -7} {
		date := time.Now().UTC().AddDate(0, 0, offset).Format("2006-01-02")
		for _, path := range []string{"/schedule?country=US&date=", "/schedule/web?date="} {
			var entries []struct {
				Show     Show `json:"show"`
				Embedded struct {
					Show Show `json:"show"`
				} `json:"_embedded"`
			}
			if e := m.get(WithInfo(ctx, Info{Trigger: "show_suggestions"}), path+date, "suggestions", 12*time.Hour, &entries); e != nil {
				lastErr = e
				continue
			}
			successes++
			for _, entry := range entries {
				show := entry.Show
				if show.ID == 0 {
					show = entry.Embedded.Show
				}
				if show.ID > 0 && show.Rating.Average >= 7 {
					show.Summary = Plain(show.Summary)
					unique[show.ID] = show
				}
			}
		}
	}
	if successes == 0 {
		return nil, lastErr
	}
	out := []SearchResult{}
	for _, show := range unique {
		out = append(out, SearchResult{Score: show.Rating.Average, Show: show})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Show.ID > out[j].Show.ID
	})
	if len(out) > 24 {
		out = out[:24]
	}
	return out, nil
}
