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
