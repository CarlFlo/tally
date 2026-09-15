package backup

import (
	"context"
	"os"
)

func (s *Service) DeleteArchive(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	archive, err := s.FindArchive(ctx, id)
	if err != nil {
		return err
	}
	path, err := s.archivePath(archive.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	if err = os.Remove(path); os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}
