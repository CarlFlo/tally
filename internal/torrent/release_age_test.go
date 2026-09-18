package torrent

import (
	"testing"
	"time"
)

func TestReleaseDelayWindowOpensWhenFirstKnownReleaseMatures(t *testing.T) {
	now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
	minimum := 20 * time.Minute
	old := EvaluateReleaseAge(now.Add(-25*time.Minute).Format(time.RFC1123Z), now, minimum)
	newer := EvaluateReleaseAge(now.Add(-2*time.Minute).Format(time.RFC1123Z), now, minimum)
	if !ReleaseDelayWindowOpen([]ReleaseAgeEvaluation{old, newer}) {
		t.Fatal("a mature first release should open the delay window for all current candidates")
	}
}

func TestReleaseDelayWindowHoldsOnlyKnownYoungReleases(t *testing.T) {
	now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
	young := EvaluateReleaseAge(now.Add(-5*time.Minute).Format(time.RFC1123Z), now, 20*time.Minute)
	if ReleaseDelayWindowOpen([]ReleaseAgeEvaluation{young}) {
		t.Fatal("young known release opened the delay window too early")
	}
	unknown := EvaluateReleaseAge("", now, 20*time.Minute)
	if !ReleaseDelayWindowOpen([]ReleaseAgeEvaluation{unknown}) {
		t.Fatal("missing release timestamps should remain neutral")
	}
}
