package settings

import (
	"encoding/json"
	"fmt"
	"github.com/CarlFlo/mediaManager/internal/config"
	"time"
	"unicode/utf8"
)

// Webhook retains the legacy settings name and URL for seamless upgrades.
type Webhook struct {
	Enabled      bool     `json:"enabled"`
	URL          string   `json:"url"`
	Type         string   `json:"type"`
	Body         string   `json:"body"`
	DiscordURL   string   `json:"discord_url"`
	BotName      string   `json:"bot_name"`
	Prefix       string   `json:"prefix"`
	Events       []string `json:"events"`
	DeliveryTime string   `json:"delivery_time"`
	Timezone     string   `json:"timezone"`
}

func (w Webhook) Defaults(defaultTimezone ...string) Webhook {
	if w.Type == "" {
		w.Type = "webhook"
	}
	if w.Body == "" {
		w.Body = `{"app":"Tally","event":"{{event}}","key":"{{key}}","level":"{{level}}","message":"{{message}}","show":"{{show}}","time":"{{time}}"}`
	}
	if w.Events == nil {
		w.Events = []string{"system_error", "job_failed"}
	}
	if w.DeliveryTime == "" {
		w.DeliveryTime = "09:00"
	}
	if w.Timezone == "" {
		w.Timezone = "UTC"
		if len(defaultTimezone) > 0 && defaultTimezone[0] != "" {
			w.Timezone = defaultTimezone[0]
		}
	}
	return w
}

func (w Webhook) Endpoint() string {
	if w.Type == "discord" {
		return w.DiscordURL
	}
	return w.URL
}

func (w Webhook) Subscribed(event string) bool {
	if !w.Enabled {
		return false
	}
	for _, e := range w.Defaults().Events {
		if e == event {
			return true
		}
	}
	return false
}

func ValidateWebhook(w Webhook, defaultTimezone ...string) error {
	w = w.Defaults(defaultTimezone...)
	if w.Type != "webhook" && w.Type != "discord" {
		return fmt.Errorf("choose Webhook or Discord")
	}
	for _, endpoint := range []string{w.URL, w.DiscordURL} {
		if endpoint != "" && (config.ValidateURL(endpoint) != nil || len(endpoint) > 4096) {
			return fmt.Errorf("enter a valid HTTP(S) notification URL")
		}
	}
	if w.Enabled && w.Endpoint() == "" {
		return fmt.Errorf("enter a notification URL")
	}
	var body map[string]any
	if len(w.Body) > 16384 || json.Unmarshal([]byte(w.Body), &body) != nil || body == nil {
		return fmt.Errorf("payload body must be a JSON object of at most 16 KB")
	}
	if err := validateTemplate(body); err != nil {
		return err
	}
	if utf8.RuneCountInString(w.BotName) > 80 || utf8.RuneCountInString(w.Prefix) > 500 {
		return fmt.Errorf("bot display name must be at most 80 characters and prefix at most 500")
	}
	if len(w.DeliveryTime) != 5 {
		return fmt.Errorf("choose a delivery time in HH:MM format")
	}
	if _, err := time.Parse("15:04", w.DeliveryTime); err != nil {
		return fmt.Errorf("choose a valid delivery time")
	}
	if _, err := time.LoadLocation(w.Timezone); err != nil {
		return fmt.Errorf("choose a valid timezone")
	}
	for _, event := range w.Events {
		switch event {
		case "system_error", "job_failed", "show_added", "show_removed", "watch_history_cleared", "episode_released":
		default:
			return fmt.Errorf("unknown notification event")
		}
	}
	return nil
}
