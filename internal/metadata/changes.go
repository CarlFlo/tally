package metadata

func (s *Service) changed(resources ...string) {
	if s.OnChange != nil {
		s.OnChange("", resources...)
	}
}

func (s *Service) changedProfile(profile string, resources ...string) {
	if s.OnChange != nil {
		s.OnChange(profile, resources...)
	}
}
