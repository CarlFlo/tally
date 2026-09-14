package jobs

func (s *Service) changed(resources ...string) {
	if s.OnChange != nil {
		s.OnChange("", resources...)
	}
}
