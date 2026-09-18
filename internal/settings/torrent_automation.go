package settings

import (
	"fmt"
	"strings"
)

const (
	TorrentQualityBest = "best"
	TorrentQuality720  = "720p"
	TorrentQuality1080 = "1080p"
	TorrentQuality2160 = "2160p"

	TorrentAutomationRulesVersion = 1
)

type TorrentAutomation struct {
	Enabled             bool     `json:"enabled"`
	PreferredQuality    string   `json:"preferred_quality"`
	MinSeeders          int      `json:"min_seeders"`
	HighConfidenceOnly  bool     `json:"high_confidence_only"`
	PreferSmaller       bool     `json:"prefer_smaller"`
	ReleaseDelayMinutes int      `json:"release_delay_minutes"`
	RetryWindowHours    int      `json:"retry_window_hours"`
	MaxCandidates       int      `json:"max_candidates"`
	RulesVersion        int      `json:"rules_version"`
	IncludeKeywords     string   `json:"include_keywords"`
	ExcludeKeywords     string   `json:"exclude_keywords"`
	AllowedGroups       []string `json:"allowed_groups"`
	PreferredGroups     []string `json:"preferred_groups"`
	AllowedUploaders    []string `json:"allowed_uploaders"`
	PreferredUploaders  []string `json:"preferred_uploaders"`
	PreferredProviders  []string `json:"preferred_providers"`
	LiveMinMBPerMinute      int `json:"live_min_mb_per_minute"`
	LiveMaxMBPerMinute      int `json:"live_max_mb_per_minute"`
	AnimatedMinMBPerMinute  int `json:"animated_min_mb_per_minute"`
	AnimatedMaxMBPerMinute  int `json:"animated_max_mb_per_minute"`
}

func DefaultTorrentAutomation() TorrentAutomation {
	return TorrentAutomation{
		PreferredQuality:    TorrentQuality1080,
		MinSeeders:          5,
		HighConfidenceOnly:  true,
		ReleaseDelayMinutes: 20,
		RetryWindowHours:    24,
		MaxCandidates:       5,
		RulesVersion:        TorrentAutomationRulesVersion,
		ExcludeKeywords:          "cam telesync hardsub dubbed",
		LiveMinMBPerMinute:       8,
		LiveMaxMBPerMinute:       220,
		AnimatedMinMBPerMinute:   4,
		AnimatedMaxMBPerMinute:   140,
	}
}

func (v TorrentAutomation) Effective() TorrentAutomation {
	defaults := DefaultTorrentAutomation()
	if v.PreferredQuality == "" {
		v.PreferredQuality = defaults.PreferredQuality
	}
	if v.ReleaseDelayMinutes == 0 {
		v.ReleaseDelayMinutes = defaults.ReleaseDelayMinutes
	}
	if v.RetryWindowHours == 0 {
		v.RetryWindowHours = defaults.RetryWindowHours
	}
	if v.MaxCandidates == 0 {
		v.MaxCandidates = defaults.MaxCandidates
	}
	if v.LiveMinMBPerMinute == 0 {
		v.LiveMinMBPerMinute = defaults.LiveMinMBPerMinute
	}
	if v.LiveMaxMBPerMinute == 0 {
		v.LiveMaxMBPerMinute = defaults.LiveMaxMBPerMinute
	}
	if v.AnimatedMinMBPerMinute == 0 {
		v.AnimatedMinMBPerMinute = defaults.AnimatedMinMBPerMinute
	}
	if v.AnimatedMaxMBPerMinute == 0 {
		v.AnimatedMaxMBPerMinute = defaults.AnimatedMaxMBPerMinute
	}
	// Existing installations predate the release-selection rules. Seed the
	// first rule set once, then preserve intentionally empty values afterwards.
	if v.RulesVersion == 0 {
		v.RulesVersion = TorrentAutomationRulesVersion
		v.IncludeKeywords = defaults.IncludeKeywords
		v.ExcludeKeywords = defaults.ExcludeKeywords
	}
	v.IncludeKeywords = normalizeKeywordText(v.IncludeKeywords)
	v.ExcludeKeywords = normalizeKeywordText(v.ExcludeKeywords)
	v.AllowedGroups = normalizeRuleList(v.AllowedGroups)
	v.PreferredGroups = normalizeRuleList(v.PreferredGroups)
	v.AllowedUploaders = normalizeRuleList(v.AllowedUploaders)
	v.PreferredUploaders = normalizeRuleList(v.PreferredUploaders)
	v.PreferredProviders = normalizeRuleList(v.PreferredProviders)
	return v
}

func ValidateTorrentAutomation(v TorrentAutomation) error {
	v = v.Effective()
	switch v.PreferredQuality {
	case TorrentQualityBest, TorrentQuality720, TorrentQuality1080, TorrentQuality2160:
	default:
		return fmt.Errorf("preferred torrent quality is invalid")
	}
	if v.MinSeeders < 0 || v.MinSeeders > 1_000_000 {
		return fmt.Errorf("minimum seeders must be between 0 and 1000000")
	}
	if v.ReleaseDelayMinutes < 0 || v.ReleaseDelayMinutes > 24*60 {
		return fmt.Errorf("release delay must be between 0 and 1440 minutes")
	}
	if v.RetryWindowHours < 1 || v.RetryWindowHours > 168 {
		return fmt.Errorf("retry window must be between 1 and 168 hours")
	}
	if v.MaxCandidates < 1 || v.MaxCandidates > 20 {
		return fmt.Errorf("candidate inspection limit must be between 1 and 20")
	}
	for label, bounds := range map[string][2]int{
		"live-action size rate": {v.LiveMinMBPerMinute, v.LiveMaxMBPerMinute},
		"animated size rate":    {v.AnimatedMinMBPerMinute, v.AnimatedMaxMBPerMinute},
	} {
		if bounds[0] < 1 || bounds[1] > 500 || bounds[0] >= bounds[1] {
			return fmt.Errorf("%s must use a 1-500 MB/min range with minimum below maximum", label)
		}
	}
	if len(v.IncludeKeywords) > 500 || len(v.ExcludeKeywords) > 500 {
		return fmt.Errorf("torrent keyword filters must be 500 characters or fewer")
	}
	for label, values := range map[string][]string{
		"release group allowlist": v.AllowedGroups,
		"preferred release groups": v.PreferredGroups,
		"uploader allowlist": v.AllowedUploaders,
		"preferred uploaders": v.PreferredUploaders,
		"preferred providers": v.PreferredProviders,
	} {
		if len(values) > 50 {
			return fmt.Errorf("%s may contain at most 50 entries", label)
		}
		for _, value := range values {
			if len(value) > 80 {
				return fmt.Errorf("%s entries must be 80 characters or fewer", label)
			}
		}
	}
	return nil
}

func normalizeKeywordText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func normalizeRuleList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}
