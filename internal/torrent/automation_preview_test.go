package torrent

import (
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/settings"
)

func TestAutomationPreviewAppliesChainFiltersAndPreferences(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	config.MinSeeders = 2
	config.AllowedGroups = []string{"FLUX"}
	config.AllowedUploaders = []string{"Trusted"}
	config.PreferredQuality = "1080p"
	config.LiveMinMBPerMinute, config.LiveMaxMBPerMinute = 10, 100
	results := []SearchResult{
		{Name: "Example.Show.S01E02.1080p.WEB-DL-FLUX", Seeders: 8, Uploader: "Trusted", Size: 2 * 1024 * 1024 * 1024},
		{Name: "Example.Show.S01E02.1080p.WEB-DL-OTHER", Seeders: 9, Uploader: "Trusted", Size: 2 * 1024 * 1024 * 1024},
		{Name: "Example.Show.S01E02.720p.WEB-DL-FLUX", Seeders: 10, Uploader: "Unknown", Size: 2 * 1024 * 1024 * 1024},
	}
	accepted, rejected := PreviewAutomationCandidates(results, EpisodeTarget{ShowTitle: "Example Show", Season: 1, Episode: 2}, config, MediaProfileLive, 45, time.Now())
	if len(accepted) != 1 || accepted[0].Name != results[0].Name || len(rejected) != 2 {
		t.Fatalf("preview rules diverged: accepted=%+v rejected=%+v", accepted, rejected)
	}
	config.LiveMaxMBPerMinute = 20
	accepted, _ = PreviewAutomationCandidates(results[:1], EpisodeTarget{ShowTitle: "Example Show", Season: 1, Episode: 2}, config, MediaProfileLive, 45, time.Now())
	if len(accepted) != 0 {
		t.Fatal("size rule was ignored")
	}
}
