package settings

import "fmt"

const (
	TorrentQualityBest  = "best"
	TorrentQuality720   = "720p"
	TorrentQuality1080  = "1080p"
	TorrentQuality2160  = "2160p"
)

type TorrentAutomation struct {
	Enabled             bool   `json:"enabled"`
	PreferredQuality    string `json:"preferred_quality"`
	MinSeeders          int    `json:"min_seeders"`
	HighConfidenceOnly  bool   `json:"high_confidence_only"`
	PreferSmaller       bool   `json:"prefer_smaller"`
	ReleaseDelayMinutes int    `json:"release_delay_minutes"`
	RetryWindowHours    int    `json:"retry_window_hours"`
	MaxCandidates       int    `json:"max_candidates"`
}

func DefaultTorrentAutomation() TorrentAutomation {
	return TorrentAutomation{
		PreferredQuality:    TorrentQuality1080,
		MinSeeders:          5,
		HighConfidenceOnly:  true,
		ReleaseDelayMinutes: 20,
		RetryWindowHours:    24,
		MaxCandidates:       5,
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
	return nil
}
