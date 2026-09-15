package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
)

func TestSuggestionsAreRankedDeduplicatedBoundedAndCached(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	control, e := providers.New(ctx, db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("date") == "" {
			t.Error("missing date bound")
		}
		entries := []any{}
		for id := 1; id <= 30; id++ {
			show := Show{ID: id, Name: "Fixture", Summary: "<p>A show</p>"}
			show.Rating.Average = 7 + float64(id)/10
			if id == 30 {
				show.Rating.Average = 6
			}
			if r.URL.Path == "/schedule/web" {
				entries = append(entries, map[string]any{"_embedded": map[string]any{"show": show}})
			} else {
				if r.URL.Path != "/schedule" || r.URL.Query().Get("country") != "US" {
					t.Error("unexpected schedule")
				}
				entries = append(entries, map[string]any{"show": show})
			}
		}
		_ = json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()
	maze := TVMaze{Control: control, BaseURL: server.URL}
	for i := 0; i < 2; i++ {
		result, e := maze.SuggestShows(ctx)
		if e != nil || len(result) != 24 {
			t.Fatalf("suggestions: %d, %v", len(result), e)
		}
		for index, entry := range result {
			if entry.Show.ID != 29-index || entry.Show.Summary != "A show" {
				t.Fatal("incorrect ranking, duplicate, or unsanitized summary")
			}
		}
	}
	if calls.Load() != 6 {
		t.Fatalf("wanted six cached schedule requests; got %d", calls.Load())
	}
}
