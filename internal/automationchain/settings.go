package automationchain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CarlFlo/tally/internal/settings"
)

const (
	DefaultLiveID     = "default-live"
	DefaultAnimatedID = "default-animated"
)

func IsDefault(id string) bool { return id == DefaultLiveID || id == DefaultAnimatedID }

func DefaultID(profile string) string {
	if profile == "animated" {
		return DefaultAnimatedID
	}
	return DefaultLiveID
}

// Apply maps editable chain controls onto the established automation policy.
// Missing controls inherit deployment settings; safety checks remain in the runner.
func Apply(base settings.TorrentAutomation, filter, selection map[string]string, profile string) (settings.TorrentAutomation, error) {
	base = base.Effective()
	for key, value := range filter {
		if value == "" && (key == "min_seeders" || key == "min_mb_per_minute" || key == "max_mb_per_minute" || key == "release_delay_minutes") {
			continue
		}
		switch key {
		case "min_seeders":
			n, err := number(value, 0, 1_000_000)
			if err != nil {
				return base, err
			}
			base.MinSeeders = n
		case "include_keywords":
			base.IncludeKeywords = value
		case "exclude_keywords":
			base.ExcludeKeywords = value
		case "allowed_groups":
			base.AllowedGroups = list(value)
		case "allowed_uploaders":
			base.AllowedUploaders = list(value)
		case "min_mb_per_minute", "max_mb_per_minute":
			n, err := number(value, 1, 500)
			if err != nil {
				return base, err
			}
			if profile == "animated" {
				if key == "min_mb_per_minute" {
					base.AnimatedMinMBPerMinute = n
				} else {
					base.AnimatedMaxMBPerMinute = n
				}
			} else {
				if key == "min_mb_per_minute" {
					base.LiveMinMBPerMinute = n
				} else {
					base.LiveMaxMBPerMinute = n
				}
			}
		case "release_delay_minutes":
			n, err := number(value, 1, 1440)
			if err != nil {
				return base, err
			}
			base.ReleaseDelayMinutes = n
		default:
			return base, fmt.Errorf("unsupported filter control %q", key)
		}
	}
	for key, value := range selection {
		if value == "" && (key == "preferred_quality" || key == "prefer_smaller" || key == "max_candidates") {
			continue
		}
		switch key {
		case "preferred_quality":
			base.PreferredQuality = value
		case "prefer_smaller":
			if value != "true" && value != "false" {
				return base, fmt.Errorf("prefer_smaller must be true or false")
			}
			base.PreferSmaller = value == "true"
		case "preferred_groups":
			base.PreferredGroups = list(value)
		case "preferred_uploaders":
			base.PreferredUploaders = list(value)
		case "preferred_providers":
			base.PreferredProviders = list(value)
		case "max_candidates":
			n, err := number(value, 1, 20)
			if err != nil {
				return base, err
			}
			base.MaxCandidates = n
		default:
			return base, fmt.Errorf("unsupported selection control %q", key)
		}
	}
	base = base.Effective()
	return base, settings.ValidateTorrentAutomation(base)
}

func number(raw string, minimum, maximum int) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < minimum || n > maximum || strconv.Itoa(n) != raw {
		return 0, fmt.Errorf("invalid chain number %q", raw)
	}
	return n, nil
}

func list(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' })
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func Join(values []string) string { return strings.Join(values, ", ") }
