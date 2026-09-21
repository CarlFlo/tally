package api

import (
	"context"
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/notifications"
	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) testNotification(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in settings.Webhook
	if err := decode(r, &in); err != nil {
		return err
	}
	in = in.Defaults(s.Config.Timezone)
	if err := settings.ValidateWebhook(in, s.Config.Timezone); err != nil {
		return bad(err.Error())
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	err := notifications.Send(ctx, s.Control, in, notifications.Message{Event: "test", Key: "test", Level: "info", Text: "Your Tally notification service is working.", Show: "Example show", Time: time.Now()})
	if err != nil {
		return remote(err)
	}
	jsonResponse(w, 200, map[string]string{"message": "Test notification delivered."})
	return nil
}
