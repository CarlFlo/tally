package torrent

import "testing"

func TestParseReleaseName(t *testing.T) {
	tests := []struct {
		name         string
		release      string
		title        string
		season       int
		episode      int
		resolution   string
		source       string
		codec        string
		multiEpisode bool
		seasonPack   bool
	}{
		{
			name:       "scene style",
			release:    "The.Last.of.Us.S03E04.1080p.WEB-DL.H264-GROUP",
			title:      "the last of us",
			season:     3,
			episode:    4,
			resolution: "1080p",
			source:     "web-dl",
			codec:      "h264",
		},
		{
			name:       "alternate episode style",
			release:    "Severance 2x10 2160p WEBRip HEVC",
			title:      "severance",
			season:     2,
			episode:    10,
			resolution: "2160p",
			source:     "webrip",
			codec:      "h265",
		},
		{
			name:         "multi episode",
			release:      "Show.S01E01E02.1080p.WEB-DL.x264-GROUP",
			title:        "show",
			season:       1,
			episode:      1,
			multiEpisode: true,
		},
		{
			name:       "season pack",
			release:    "Show.S01.1080p.BluRay.x265-GROUP",
			title:      "show",
			season:     1,
			seasonPack: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed := ParseReleaseName(test.release)
			if parsed.Title != test.title || parsed.Season != test.season || parsed.Episode != test.episode || parsed.Resolution != test.resolution || parsed.Source != test.source || parsed.Codec != test.codec || parsed.MultiEpisode != test.multiEpisode || parsed.SeasonPack != test.seasonPack {
				t.Fatalf("ParseReleaseName(%q) = %+v", test.release, parsed)
			}
		})
	}
}

func TestEvaluateSearchCandidateExactEpisodeIsHighConfidence(t *testing.T) {
	candidate := SearchResult{
		Name:     "The.Last.of.Us.S03E04.1080p.WEB-DL.H264-GROUP",
		Seeders:  120,
		TVMazeID: "100",
		URL:      "https://jackett.invalid/download/one",
	}
	target := EpisodeTarget{ShowTitle: "The Last of Us", Season: 3, Episode: 4, TVMazeID: "100"}
	assessment := EvaluateSearchCandidate(candidate, target)
	if assessment.Confidence != ConfidenceHigh || assessment.Verification != VerificationUnverified || assessment.Rejected() {
		t.Fatalf("unexpected assessment: %+v", assessment)
	}
	if !hasReason(assessment.Reasons, ReasonShowMatch) || !hasReason(assessment.Reasons, ReasonEpisodeMatch) {
		t.Fatalf("expected show and episode match reasons: %+v", assessment.Reasons)
	}
}

func TestEvaluateSearchCandidateUsesAliasAndYear(t *testing.T) {
	candidate := SearchResult{Name: "The.Office.2005.S02E03.720p.HDTV.x264-GROUP", Seeders: 10}
	target := EpisodeTarget{ShowTitle: "The Office (US)", Aliases: []string{"The Office"}, Year: 2005, Season: 2, Episode: 3}
	assessment := EvaluateSearchCandidate(candidate, target)
	if assessment.Confidence != ConfidenceHigh {
		t.Fatalf("alias/year match should be high confidence: %+v", assessment)
	}
}

func TestEvaluateSearchCandidateRejectsWrongEpisode(t *testing.T) {
	assessment := EvaluateSearchCandidate(
		SearchResult{Name: "Show.S01E03.1080p.WEB-DL.x264-GROUP"},
		EpisodeTarget{ShowTitle: "Show", Season: 1, Episode: 2},
	)
	if assessment.Confidence != ConfidenceRejected || !hasReason(assessment.HardRejections, ReasonWrongEpisode) {
		t.Fatalf("wrong episode was not rejected: %+v", assessment)
	}
}

func TestEvaluateSearchCandidateRejectsConflictingExternalID(t *testing.T) {
	assessment := EvaluateSearchCandidate(
		SearchResult{Name: "Show.S01E02.1080p.WEB-DL.x264-GROUP", TVMazeID: "999"},
		EpisodeTarget{ShowTitle: "Show", Season: 1, Episode: 2, TVMazeID: "100"},
	)
	if assessment.Confidence != ConfidenceRejected || !hasReason(assessment.HardRejections, ReasonWrongShow) {
		t.Fatalf("conflicting external ID was not rejected: %+v", assessment)
	}
}

func TestEvaluateSearchCandidateKeepsSpecialsReviewOnly(t *testing.T) {
	assessment := EvaluateSearchCandidate(
		SearchResult{Name: "Show.S00E05.1080p.WEB-DL.x264-GROUP"},
		EpisodeTarget{ShowTitle: "Show", Season: 0, Episode: 5},
	)
	if assessment.Confidence != ConfidenceMedium || assessment.Rejected() {
		t.Fatalf("special should require review without being rejected: %+v", assessment)
	}
}

func TestEvaluateSearchCandidateRejectsMultiEpisodeAndSeasonPack(t *testing.T) {
	for _, release := range []string{
		"Show.S01E01E02.1080p.WEB-DL.x264-GROUP",
		"Show.S01.1080p.BluRay.x265-GROUP",
	} {
		assessment := EvaluateSearchCandidate(SearchResult{Name: release}, EpisodeTarget{ShowTitle: "Show", Season: 1, Episode: 1})
		if assessment.Confidence != ConfidenceRejected || !hasReason(assessment.HardRejections, ReasonUnsupportedPack) {
			t.Fatalf("unsupported pack %q was not rejected: %+v", release, assessment)
		}
	}
}
