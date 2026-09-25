package automationchain

import (
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
)

func TestApplyProfileFiltersAndSelection(t *testing.T) {
	base := settings.DefaultTorrentAutomation()
	filter := map[string]string{
		"min_seeders": "2", "include_keywords": "proper", "exclude_keywords": "",
		"allowed_groups": "FLUX, NTb", "allowed_uploaders": "Trusted",
		"min_mb_per_minute": "5", "max_mb_per_minute": "100", "release_delay_minutes": "1",
	}
	selection := map[string]string{
		"preferred_quality": "2160p", "prefer_smaller": "true", "preferred_groups": "FLUX",
		"preferred_uploaders": "Trusted", "preferred_providers": "Indexer A", "max_candidates": "2",
	}
	got, err := Apply(base, filter, selection, "animated")
	if err != nil {
		t.Fatal(err)
	}
	if got.MinSeeders != 2 || got.IncludeKeywords != "proper" || got.ExcludeKeywords != "" || got.AnimatedMinMBPerMinute != 5 || got.AnimatedMaxMBPerMinute != 100 || got.LiveMinMBPerMinute != base.LiveMinMBPerMinute || got.PreferredQuality != "2160p" || !got.PreferSmaller || got.MaxCandidates != 2 || len(got.AllowedGroups) != 2 || got.PreferredProviders[0] != "Indexer A" {
		t.Fatalf("chain rules were not applied: %+v", got)
	}
	if _, err := Apply(base, map[string]string{"min_mb_per_minute": "300"}, nil, "live"); err == nil {
		t.Fatal("accepted inverted size range")
	}
	if _, err := Apply(base, map[string]string{"allowed_uploaders": ""}, map[string]string{"preferred_quality": "invalid"}, "live"); err == nil {
		t.Fatal("accepted invalid quality")
	}
}
