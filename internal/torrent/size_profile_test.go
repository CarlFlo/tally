package torrent

import (
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
)

func TestSizeProfileUsesSeparateLiveAndAnimatedRanges(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	size := int64(6 * 1024 * 1024 * 1024)
	live := EvaluateSizeProfile(size, 45, config, MediaProfileLive)
	animated := EvaluateSizeProfile(size, 45, config, MediaProfileAnimated)
	if !live.Known || !live.InActiveRange {
		t.Fatalf("expected live-action fixture inside range: %+v", live)
	}
	if !animated.Known || !animated.InActiveRange {
		t.Fatalf("expected animated fixture inside range: %+v", animated)
	}
	if live.ActiveScore != live.LiveScore || animated.ActiveScore != animated.AnimatedScore {
		t.Fatalf("active score did not follow media profile: live=%+v animated=%+v", live, animated)
	}
	if live.LiveScore == live.AnimatedScore {
		t.Fatalf("debug scores should differ for the same bitrate profile: %+v", live)
	}
}

func TestSizeProfileFiltersOnlyKnownOutliers(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	tooSmall := EvaluateSizeProfile(20*1024*1024, 45, config, MediaProfileLive)
	if !tooSmall.Known || tooSmall.InActiveRange {
		t.Fatalf("ridiculously small release was not filtered: %+v", tooSmall)
	}
	unknownRuntime := EvaluateSizeProfile(20*1024*1024, 0, config, MediaProfileLive)
	if unknownRuntime.Known || !unknownRuntime.InActiveRange {
		t.Fatalf("missing runtime should stay neutral: %+v", unknownRuntime)
	}
	unknownSize := EvaluateSizeProfile(0, 45, config, MediaProfileAnimated)
	if unknownSize.Known || !unknownSize.InActiveRange {
		t.Fatalf("missing size should stay neutral: %+v", unknownSize)
	}
}

func TestSizeProfileHonorsConfiguredRangeBoundaries(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	config.LiveMinMBPerMinute = 10
	config.LiveMaxMBPerMinute = 20
	size := int64(15 * 60 * 1024 * 1024)
	result := EvaluateSizeProfile(size, 60, config, MediaProfileLive)
	if !result.InActiveRange || result.ActiveScore != 100 {
		t.Fatalf("midpoint should be ideal: %+v", result)
	}
}
