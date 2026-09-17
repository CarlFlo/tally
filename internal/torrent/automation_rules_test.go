package torrent

import (
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
)

func TestAutomationKeywordAndAllowlistRules(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	config.IncludeKeywords = "web proper"
	config.ExcludeKeywords = "dubbed hardsub"
	config.AllowedGroups = []string{"FLUX", "NTb"}
	config.AllowedUploaders = []string{"trusted-user"}

	if !automationKeywordsMatch("Example.Show.S01E02.1080p.WEB.PROPER-FLUX", config) {
		t.Fatal("expected include keywords to match")
	}
	if automationKeywordsMatch("Example.Show.S01E02.1080p.WEB.PROPER.DUBBED-FLUX", config) {
		t.Fatal("exclude keyword was ignored")
	}
	if !automationGroupAllowed("flux", config) || automationGroupAllowed("OTHER", config) {
		t.Fatal("release group allowlist is not case-insensitive and restrictive")
	}
	if !automationUploaderAllowed("Trusted-User", config) || automationUploaderAllowed("", config) || automationUploaderAllowed("other", config) {
		t.Fatal("uploader allowlist did not handle present/missing metadata correctly")
	}
}

func TestAutomationPreferenceSignalsRankTrustedSourcesBeforeQuality(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	config.PreferredQuality = settings.TorrentQuality1080
	config.PreferredGroups = []string{"FLUX"}
	config.PreferredUploaders = []string{"trusted-user"}
	config.PreferredProviders = []string{"Trusted Indexer"}

	trusted := automationCandidate{
		Result: SearchResult{Name: "Example.Show.S01E02.720p.WEB-DL-FLUX", Provider: "Trusted Indexer", Uploader: "trusted-user", Seeders: 20},
		Assessment: ReleaseAssessment{Confidence: ConfidenceHigh, Parsed: ParseReleaseName("Example.Show.S01E02.720p.WEB-DL-FLUX")},
	}
	quality := automationCandidate{
		Result: SearchResult{Name: "Example.Show.S01E02.1080p.WEB-DL-OTHER", Provider: "Other Indexer", Seeders: 200},
		Assessment: ReleaseAssessment{Confidence: ConfidenceHigh, Parsed: ParseReleaseName("Example.Show.S01E02.1080p.WEB-DL-OTHER")},
	}
	items := []automationCandidate{quality, trusted}
	rankAutomationCandidates(items, config)
	if items[0].Result.Provider != "Trusted Indexer" {
		t.Fatalf("preferred source did not materially affect ranking: %+v", items)
	}
	signals := AutomationPreferenceSignals(trusted.Result, trusted.Assessment.Parsed, config)
	if !signals.PreferredGroup || !signals.PreferredUploader || !signals.PreferredProvider {
		t.Fatalf("expected all preference signals, got %+v", signals)
	}
}

func TestTrustedSourceDoesNotOverrideWrongEpisodeConfidence(t *testing.T) {
	config := settings.DefaultTorrentAutomation()
	config.PreferredGroups = []string{"FLUX"}
	config.PreferredUploaders = []string{"trusted-user"}
	config.PreferredProviders = []string{"Trusted Indexer"}
	result := SearchResult{
		Name: "Example.Show.S01E03.1080p.WEB-DL-FLUX",
		Provider: "Trusted Indexer",
		Uploader: "trusted-user",
		Seeders: 500,
	}
	assessment := EvaluateSearchCandidate(result, EpisodeTarget{ShowTitle: "Example Show", Season: 1, Episode: 2})
	if assessment.Confidence != ConfidenceRejected {
		t.Fatalf("trusted source changed wrong-episode confidence: %+v", assessment)
	}
	if automationPreferenceScore(result, assessment.Parsed, config) == 0 {
		t.Fatal("test fixture did not carry a preference signal")
	}
}
