package torrent

import (
	"strconv"
	"strings"
)

type EpisodeTarget struct {
	ShowTitle string
	Aliases   []string
	Year      int
	Season    int
	Episode   int
	RuntimeMinutes int
	TVDBID    string
	TMDBID    string
	IMDBID    string
	TVMazeID  string
}

func EvaluateSearchCandidate(candidate SearchResult, target EpisodeTarget) ReleaseAssessment {
	parsed := ParseReleaseName(candidate.Name)
	assessment := ReleaseAssessment{
		Confidence:   ConfidenceLow,
		Verification: VerificationUnverified,
		Parsed:       parsed,
		InfoHash:     candidate.InfoHash,
	}

	showMatch, showConflict := candidateMatchesTargetShow(candidate, parsed, target)
	if showConflict {
		assessment.HardRejections = append(assessment.HardRejections, AssessmentReason{Code: ReasonWrongShow})
	} else if showMatch {
		assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonShowMatch})
		assessment.Score += 55
	} else {
		assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonAmbiguousIdentity})
	}

	if parsed.MultiEpisode || parsed.SeasonPack {
		assessment.HardRejections = append(assessment.HardRejections, AssessmentReason{Code: ReasonUnsupportedPack})
	} else if parsed.Season > 0 || parsed.Episode > 0 || target.Season == 0 {
		if parsed.Season == target.Season && parsed.Episode == target.Episode {
			assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonEpisodeMatch})
			assessment.Score += 35
		} else {
			assessment.HardRejections = append(assessment.HardRejections, AssessmentReason{Code: ReasonWrongEpisode})
		}
	} else {
		assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonAmbiguousIdentity, Detail: "episode_not_parsed"})
	}

	if candidate.Seeders > 0 {
		assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonHealthySwarm})
		assessment.Score += 5
	}
	if parsed.Resolution != "" {
		assessment.Score += 2
	}
	if parsed.Source != "" {
		assessment.Score += 2
	}
	if parsed.Codec != "" {
		assessment.Score++
	}

	if candidate.Magnet != "" && candidate.URL == "" {
		assessment.Reasons = append(assessment.Reasons, AssessmentReason{Code: ReasonUnverifiableMagnet})
	}

	if len(assessment.HardRejections) > 0 {
		assessment.Confidence = ConfidenceRejected
		return assessment
	}

	episodeMatch := hasReason(assessment.Reasons, ReasonEpisodeMatch)
	switch {
	case showMatch && episodeMatch && target.Season != 0:
		assessment.Confidence = ConfidenceHigh
	case showMatch && episodeMatch:
		// Specials remain review-only in the first automation version.
		assessment.Confidence = ConfidenceMedium
	case episodeMatch:
		assessment.Confidence = ConfidenceMedium
	default:
		assessment.Confidence = ConfidenceLow
	}
	return assessment
}

func candidateMatchesTargetShow(candidate SearchResult, parsed ParsedRelease, target EpisodeTarget) (match bool, conflict bool) {
	matchedExternalID := false
	for _, pair := range [][2]string{
		{candidate.TVDBID, target.TVDBID},
		{candidate.TMDBID, target.TMDBID},
		{candidate.IMDBID, target.IMDBID},
		{candidate.TVMazeID, target.TVMazeID},
	} {
		candidateID, targetID := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])
		if candidateID == "" || targetID == "" {
			continue
		}
		if !strings.EqualFold(candidateID, targetID) {
			return false, true
		}
		matchedExternalID = true
	}
	if matchedExternalID {
		return true, false
	}

	// A release-title mismatch alone is not authoritative enough to call this a
	// different show. Release naming is inconsistent across indexers, aliases,
	// punctuation, and localized titles. Keep it ambiguous for ranking/manual
	// review; only conflicting external IDs are a hard metadata-level conflict.
	return releaseTitleMatchesTarget(parsed.Title, target), false
}

func releaseTitleMatchesTarget(title string, target EpisodeTarget) bool {
	candidateTitle := normalizeReleaseTitle(title)
	if candidateTitle == "" {
		return false
	}
	names := append([]string{target.ShowTitle}, target.Aliases...)
	for _, name := range names {
		normalized := normalizeReleaseTitle(name)
		if normalized == "" {
			continue
		}
		if candidateTitle == normalized {
			return true
		}
		if target.Year > 0 && candidateTitle == normalized+" "+strconv.Itoa(target.Year) {
			return true
		}
	}
	return false
}

func hasReason(reasons []AssessmentReason, code AssessmentReasonCode) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
