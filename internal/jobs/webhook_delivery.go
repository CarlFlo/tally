package jobs

import "github.com/CarlFlo/mediaManager/internal/notifications"

func (s *Service) deliverNotifications() {
	defer s.wg.Done()
	service := notifications.Service{DB: s.DB, Requester: s.Control, Timezone: s.Config.Timezone, OnChange: s.OnChange}
	service.Run(s.ctx, s.notifications)
}
