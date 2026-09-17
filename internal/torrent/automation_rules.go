package torrent

import (
	"strings"

	"github.com/CarlFlo/tally/internal/settings"
)

type PreferenceSignals struct {
	PreferredGroup    bool `json:"preferred_group"`
	PreferredUploader bool `json:"preferred_uploader"`
	PreferredProvider bool `json:"preferred_provider"`
}

func AutomationPreferenceSignals(result SearchResult, parsed ParsedRelease, config settings.TorrentAutomation) PreferenceSignals {
	return PreferenceSignals{
		PreferredGroup:    ruleListContains(config.PreferredGroups, parsed.Group),
		PreferredUploader: ruleListContains(config.PreferredUploaders, result.Uploader),
		PreferredProvider: ruleListContains(config.PreferredProviders, result.Provider),
	}
}

func automationPreferenceScore(result SearchResult, parsed ParsedRelease, config settings.TorrentAutomation) int {
	signals := AutomationPreferenceSignals(result, parsed, config)
	score := 0
	if signals.PreferredUploader {
		score += 4
	}
	if signals.PreferredProvider {
		score += 3
	}
	if signals.PreferredGroup {
		score += 2
	}
	return score
}

func automationKeywordsMatch(name string, config settings.TorrentAutomation) bool {
	name = strings.ToLower(name)
	for _, word := range strings.Fields(strings.ToLower(config.IncludeKeywords)) {
		if !strings.Contains(name, word) {
			return false
		}
	}
	for _, word := range strings.Fields(strings.ToLower(config.ExcludeKeywords)) {
		if strings.Contains(name, word) {
			return false
		}
	}
	return true
}

func automationGroupAllowed(group string, config settings.TorrentAutomation) bool {
	return len(config.AllowedGroups) == 0 || ruleListContains(config.AllowedGroups, group)
}

func automationUploaderAllowed(uploader string, config settings.TorrentAutomation) bool {
	return len(config.AllowedUploaders) == 0 || ruleListContains(config.AllowedUploaders, uploader)
}

func ruleListContains(values []string, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), candidate) {
			return true
		}
	}
	return false
}
