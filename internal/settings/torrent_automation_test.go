package settings

import "testing"

func TestTorrentAutomationRuleDefaultsAreSeededOnce(t *testing.T) {
	legacy := TorrentAutomation{PreferredQuality: TorrentQuality1080, RetryWindowHours: 24, MaxCandidates: 5}
	effective := legacy.Effective()
	if effective.RulesVersion != TorrentAutomationRulesVersion || effective.ExcludeKeywords == "" {
		t.Fatalf("legacy settings did not receive rule defaults: %+v", effective)
	}

	cleared := effective
	cleared.IncludeKeywords = ""
	cleared.ExcludeKeywords = ""
	cleared.AllowedGroups = nil
	cleared.PreferredGroups = nil
	cleared.AllowedUploaders = nil
	cleared.PreferredUploaders = nil
	cleared.PreferredProviders = nil
	after := cleared.Effective()
	if after.ExcludeKeywords != "" || after.IncludeKeywords != "" {
		t.Fatalf("intentional empty filters were replaced by defaults: %+v", after)
	}
}

func TestTorrentAutomationRuleListsAreNormalized(t *testing.T) {
	config := DefaultTorrentAutomation()
	config.AllowedGroups = []string{" FLUX ", "flux", "", "NTb"}
	config.PreferredProviders = []string{" Indexer A ", "indexer a", "Indexer B"}
	effective := config.Effective()
	if len(effective.AllowedGroups) != 2 || effective.AllowedGroups[0] != "FLUX" || effective.AllowedGroups[1] != "NTb" {
		t.Fatalf("unexpected normalized groups: %#v", effective.AllowedGroups)
	}
	if len(effective.PreferredProviders) != 2 {
		t.Fatalf("unexpected normalized providers: %#v", effective.PreferredProviders)
	}
}

func TestTorrentAutomationValidatesSizeProfileRanges(t *testing.T) {
	config := DefaultTorrentAutomation()
	if err := ValidateTorrentAutomation(config); err != nil {
		t.Fatalf("defaults should be valid: %v", err)
	}
	config.LiveMinMBPerMinute = config.LiveMaxMBPerMinute
	if err := ValidateTorrentAutomation(config); err == nil {
		t.Fatal("equal live-action bounds were accepted")
	}
	config = DefaultTorrentAutomation()
	config.AnimatedMaxMBPerMinute = 501
	if err := ValidateTorrentAutomation(config); err == nil {
		t.Fatal("animated range above safety maximum was accepted")
	}
}

func TestTorrentAutomationRequestPolicyDefaultsAreSeededOnce(t *testing.T) {
	legacy := TorrentAutomation{PreferredQuality: TorrentQuality1080, RetryWindowHours: 24, MaxCandidates: 5}
	effective := legacy.Effective()
	if effective.RequestPolicyVersion != TorrentAutomationRequestPolicyVersion {
		t.Fatalf("legacy request policy was not versioned: %+v", effective)
	}
	if effective.DiscoveryBudget != 5 || effective.RetryFirstMinutes != 30 || effective.RetrySecondMinutes != 120 || effective.RetryLaterMinutes != 360 || !effective.PrioritizeRecent {
		t.Fatalf("legacy settings did not receive safe request defaults: %+v", effective)
	}

	effective.PrioritizeRecent = false
	after := effective.Effective()
	if after.PrioritizeRecent {
		t.Fatalf("intentional recency-priority disable was overwritten: %+v", after)
	}
}

func TestTorrentAutomationValidatesRequestRestraint(t *testing.T) {
	config := DefaultTorrentAutomation()
	if err := ValidateTorrentAutomation(config); err != nil {
		t.Fatalf("defaults should be valid: %v", err)
	}
	config.DiscoveryBudget = 26
	if err := ValidateTorrentAutomation(config); err == nil {
		t.Fatal("oversized discovery budget was accepted")
	}
	config = DefaultTorrentAutomation()
	config.RetryFirstMinutes = 180
	config.RetrySecondMinutes = 60
	if err := ValidateTorrentAutomation(config); err == nil {
		t.Fatal("decreasing retry backoff was accepted")
	}
}

func TestTorrentAutomationCompletionThresholdDefaultsAndValidates(t *testing.T) {
	legacy := TorrentAutomation{PreferredQuality: TorrentQuality1080, RetryWindowHours: 24, MaxCandidates: 5}
	if effective := legacy.Effective(); effective.CompletionPercent != 100 {
		t.Fatalf("legacy completion threshold was not conservative: %+v", effective)
	}
	config := DefaultTorrentAutomation()
	config.CompletionPercent = 101
	if err := ValidateTorrentAutomation(config); err == nil {
		t.Fatal("completion threshold above 100 was accepted")
	}
}
