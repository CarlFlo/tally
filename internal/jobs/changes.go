package jobs

func (s *Service) changed() {
	if s.OnChange != nil {
		s.OnChange()
	}
}
