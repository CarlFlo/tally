package metadata

func (s *Service) changed() {
	if s.OnChange != nil {
		s.OnChange()
	}
}
