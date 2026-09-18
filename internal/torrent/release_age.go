package torrent

import (
	"net/mail"
	"strings"
	"time"
)

type ReleaseAgeEvaluation struct {
	Known      bool `json:"known"`
	AgeMinutes int  `json:"age_minutes,omitempty"`
	Ready      bool `json:"ready"`
}

func EvaluateReleaseAge(published string, now time.Time, minimum time.Duration) ReleaseAgeEvaluation {
	out := ReleaseAgeEvaluation{Ready: true}
	published = strings.TrimSpace(published)
	if published == "" {
		return out
	}
	parsed, err := mail.ParseDate(published)
	if err != nil {
		fallback, fallbackErr := time.Parse(time.RFC3339, published)
		if fallbackErr != nil {
			return out
		}
		parsed = fallback
	}
	out.Known = true
	age := now.Sub(parsed)
	if age > 0 {
		out.AgeMinutes = int(age / time.Minute)
	}
	out.Ready = minimum <= 0 || age >= minimum
	return out
}

func ReleaseDelayWindowOpen(ages []ReleaseAgeEvaluation) bool {
	known := false
	for _, age := range ages {
		if !age.Known {
			continue
		}
		known = true
		if age.Ready {
			return true
		}
	}
	return !known
}
