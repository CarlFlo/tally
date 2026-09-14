package settings

import "testing"

func TestJackettSettingsValidation(t *testing.T) {
	valid := Search{BaseURL: "http://jackett:9117", APIKey: "secret", Enabled: true}
	if err := ValidateSearch(valid); err != nil || !valid.Configured() {
		t.Fatalf("valid Jackett settings rejected: %v", err)
	}
	invalid := []Search{
		{BaseURL: "file:///config", APIKey: "secret", Enabled: true},
		{BaseURL: "http://user:pass@jackett:9117", APIKey: "secret", Enabled: true},
		{BaseURL: "http://jackett:9117?key=leak", APIKey: "secret", Enabled: true},
		{BaseURL: "http://jackett:9117", APIKey: "", Enabled: true},
		{BaseURL: "http://jackett:9117", APIKey: "bad\nkey", Enabled: true},
	}
	for _, candidate := range invalid {
		if err := ValidateSearch(candidate); err == nil {
			t.Fatalf("invalid Jackett settings accepted: %+v", candidate)
		}
	}
	if err := ValidateSearch(Search{}); err != nil {
		t.Fatalf("empty disabled settings rejected: %v", err)
	}
}

func TestLegacyJackettEndpointBecomesBaseURL(t *testing.T) {
	legacy := Search{Providers: []Indexer{{URL: "http://jackett:9117/api/v2.0/indexers/all/results/torznab/api", APIKey: "secret", Enabled: true}}}
	effective := legacy.Effective()
	if effective.BaseURL != "http://jackett:9117" || !effective.Configured() {
		t.Fatalf("legacy Jackett connection was not preserved: %+v", effective)
	}
}
