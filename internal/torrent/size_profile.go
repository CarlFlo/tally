package torrent

import (
	"math"

	"github.com/CarlFlo/tally/internal/settings"
)

type SizeProfileEvaluation struct {
	Known          bool    `json:"known"`
	RuntimeMinutes int     `json:"runtime_minutes,omitempty"`
	MBPerMinute    float64 `json:"mb_per_minute,omitempty"`
	ActiveProfile  string  `json:"active_profile"`
	ActiveScore    int     `json:"active_score,omitempty"`
	LiveScore      int     `json:"live_score,omitempty"`
	AnimatedScore  int     `json:"animated_score,omitempty"`
	InActiveRange  bool    `json:"in_active_range"`
}

func EvaluateSizeProfile(sizeBytes int64, runtimeMinutes int, config settings.TorrentAutomation, activeProfile string) SizeProfileEvaluation {
	config = config.Effective()
	out := SizeProfileEvaluation{RuntimeMinutes: runtimeMinutes, ActiveProfile: activeProfile, InActiveRange: true}
	if activeProfile != MediaProfileAnimated {
		out.ActiveProfile = MediaProfileLive
	}
	if sizeBytes <= 0 || runtimeMinutes <= 0 {
		return out
	}
	out.Known = true
	out.MBPerMinute = float64(sizeBytes) / (1024 * 1024) / float64(runtimeMinutes)
	out.LiveScore = sizeRangeScore(out.MBPerMinute, float64(config.LiveMinMBPerMinute), float64(config.LiveMaxMBPerMinute))
	out.AnimatedScore = sizeRangeScore(out.MBPerMinute, float64(config.AnimatedMinMBPerMinute), float64(config.AnimatedMaxMBPerMinute))
	if out.ActiveProfile == MediaProfileAnimated {
		out.ActiveScore = out.AnimatedScore
		out.InActiveRange = out.MBPerMinute >= float64(config.AnimatedMinMBPerMinute) && out.MBPerMinute <= float64(config.AnimatedMaxMBPerMinute)
	} else {
		out.ActiveScore = out.LiveScore
		out.InActiveRange = out.MBPerMinute >= float64(config.LiveMinMBPerMinute) && out.MBPerMinute <= float64(config.LiveMaxMBPerMinute)
	}
	return out
}

func sizeRangeScore(value, minimum, maximum float64) int {
	if minimum <= 0 || maximum <= minimum || value <= 0 {
		return 0
	}
	midpoint := (minimum + maximum) / 2
	half := (maximum - minimum) / 2
	distance := math.Abs(value - midpoint)
	var score float64
	if distance <= half {
		score = 100 - 50*(distance/half)
	} else {
		score = 50 - 50*((distance-half)/half)
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}
