package notifications

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/settings"
)

type Message struct {
	Event string
	Key   string
	Level string
	Text  string
	Show  string
	Time  time.Time
}

func Payload(config settings.Webhook, message Message) ([]byte, error) {
	config = config.Defaults()
	if config.Type == "discord" {
		content := strings.TrimSpace(config.Prefix + " " + message.Text)
		runes := []rune(content)
		if len(runes) > 2000 {
			content = string(runes[:1999]) + "\u2026"
		}
		body := map[string]any{"content": content, "allowed_mentions": map[string]any{"parse": []string{}}}
		if config.BotName != "" {
			body["username"] = config.BotName
		}
		return json.Marshal(body)
	}
	var body any
	if err := json.Unmarshal([]byte(config.Body), &body); err != nil {
		return nil, err
	}
	replace := strings.NewReplacer("{{event}}", message.Event, "{{key}}", message.Key, "{{level}}", message.Level, "{{message}}", message.Text, "{{show}}", message.Show, "{{time}}", message.Time.UTC().Format(time.RFC3339))
	return json.Marshal(expand(body, replace))
}

// Expand decoded string values before encoding: names/messages cannot inject JSON.
func expand(value any, replace *strings.Replacer) any {
	switch v := value.(type) {
	case string:
		return replace.Replace(v)
	case []any:
		for i, x := range v {
			v[i] = expand(x, replace)
		}
	case map[string]any:
		for k, x := range v {
			v[k] = expand(x, replace)
		}
	}
	return value
}
