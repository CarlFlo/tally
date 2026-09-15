package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/notifications"
)

func (s *Server) notificationTimePreview(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	var input struct {
		DeliveryTime string `json:"delivery_time"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	if _, err := time.Parse("15:04", input.DeliveryTime); err != nil || len(input.DeliveryTime) != 5 {
		return bad("choose a valid delivery time")
	}
	location, err := time.LoadLocation(s.Config.Timezone)
	if err != nil {
		return apiError{500, "server timezone is invalid"}
	}
	next := notifications.DeliveryAt(time.Now(), input.DeliveryTime, location)
	jsonResponse(w, http.StatusOK, map[string]any{
		"next_delivery":   next.Unix(),
		"server_timezone": s.Config.Timezone,
	})
	return nil
}
