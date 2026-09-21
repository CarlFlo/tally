package settings

import "strings"

func RedactSearchSecrets(value Search) (Search, map[string]bool) {
	value = value.Effective()
	configured := map[string]bool{"api_key": value.APIKey != ""}
	value.APIKey = ""
	return value, configured
}

func MergeSearchSecrets(next, current Search) Search {
	current = current.Effective()
	if next.APIKey == "" && strings.TrimSpace(next.BaseURL) != "" {
		next.APIKey = current.APIKey
	}
	return next
}

func RedactWebhookSecrets(value Webhook, defaultTimezone string) (Webhook, map[string]bool) {
	value = value.Defaults(defaultTimezone)
	configured := map[string]bool{
		"url":         value.URL != "",
		"discord_url": value.DiscordURL != "",
	}
	value.URL = ""
	value.DiscordURL = ""
	return value, configured
}

func MergeWebhookSecrets(next, current Webhook, clear ...map[string]bool) Webhook {
	clearSecrets := map[string]bool{}
	if len(clear) > 0 && clear[0] != nil {
		clearSecrets = clear[0]
	}
	if next.URL == "" && !clearSecrets["url"] {
		next.URL = current.URL
	}
	if next.DiscordURL == "" && !clearSecrets["discord_url"] {
		next.DiscordURL = current.DiscordURL
	}
	return next
}
