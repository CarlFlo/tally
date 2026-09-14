package metadata

import (
	"time"
)

func NextCheck(show Show, episodes []Episode, now time.Time) time.Time {
	if show.Status == "Ended" {
		return now.Add(30 * 24 * time.Hour)
	}
	soon := time.Time{}
	for _, ep := range episodes {
		t, e := time.Parse(time.RFC3339, ep.Airstamp)
		if e == nil && t.After(now) && (soon.IsZero() || t.Before(soon)) {
			soon = t
		}
	}
	if !soon.IsZero() {
		if soon.Sub(now) < 7*24*time.Hour {
			return now.Add(6 * time.Hour)
		}
		return now.Add(48 * time.Hour)
	}
	if show.Status == "To Be Determined" {
		return now.Add(3 * 24 * time.Hour)
	}
	return now.Add(24 * time.Hour)
}
