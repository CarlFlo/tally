package notifications

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

type recorder struct {
	requests []providers.Request
	status   int
	hook     func()
}

func (r *recorder) Do(_ context.Context, in providers.Request) (providers.Response, error) {
	r.requests = append(r.requests, in)
	if r.hook != nil {
		hook := r.hook
		r.hook = nil
		hook()
	}
	status := r.status
	if status == 0 {
		status = 204
	}
	return providers.Response{Status: status}, nil
}

func TestWebhookTemplateEscapesMessageAndDiscordConfirmsDelivery(t *testing.T) {
	config := settings.Webhook{URL: "http://fixture.invalid", Body: `{"nested":{"text":"{{message}}"},"show":"{{show}}","array":["{{event}}",3]}`}
	if err := settings.ValidateWebhook(config); err != nil {
		t.Fatal(err)
	}
	message := Message{Event: "episode_released", Text: "Quoted \"show\"\nline two", Show: "a\\b", Time: time.Now()}
	raw, err := Payload(config, message)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["nested"].(map[string]any)["text"] != message.Text {
		t.Fatal("template changed message")
	}
	r := &recorder{}
	config = settings.Webhook{Type: "discord", DiscordURL: "http://fixture.invalid/webhook?thread_id=123", BotName: "Tally", Prefix: "TV:"}
	if err = Send(context.Background(), r, config, message); err != nil {
		t.Fatal(err)
	}
	req := r.requests[0]
	if req.URL != "http://fixture.invalid/webhook?thread_id=123&wait=true" || !req.NoRetry || req.Provider != "webhook" {
		t.Fatal("incorrect provider policy", req)
	}
	if err = json.Unmarshal(req.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["username"] != "Tally" || body["content"] != "TV: "+message.Text || len(body["allowed_mentions"].(map[string]any)["parse"].([]any)) != 0 {
		t.Fatal("incorrect Discord payload", body)
	}
	r.status = 401
	if err = Send(context.Background(), r, config, message); err == nil {
		t.Fatal("rejected credentials reported success")
	}
	config.Body = `{"message":"{{unknown}}"}`
	if settings.ValidateWebhook(config) == nil {
		t.Fatal("unknown placeholder accepted")
	}
}
