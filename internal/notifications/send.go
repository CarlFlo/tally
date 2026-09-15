package notifications

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

func Send(ctx context.Context, requester providers.Requester, config settings.Webhook, message Message) error {
	config = config.Defaults()
	if err := settings.ValidateWebhook(config); err != nil {
		return err
	}
	if config.Endpoint() == "" {
		return fmt.Errorf("enter a notification URL")
	}
	payload, err := Payload(config, message)
	if err != nil {
		return err
	}
	endpoint := config.Endpoint()
	if config.Type == "discord" {
		u, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("invalid Discord URL")
		}
		query := u.Query()
		query.Set("wait", "true")
		u.RawQuery = query.Encode()
		endpoint = u.String()
	}
	response, err := requester.Do(ctx, providers.Request{Provider: "webhook", URL: endpoint, Method: "POST", Body: payload, Header: http.Header{"Content-Type": []string{"application/json"}}, Trigger: "notification", NoRetry: true, MaxBytes: 1 << 20})
	if err != nil {
		return fmt.Errorf("notification service could not be reached; check the URL and connection")
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("notification service returned HTTP %d", response.Status)
	}
	return nil
}
