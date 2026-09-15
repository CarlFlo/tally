package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
	"strings"
)

type Archive struct {
	ID        string
	Filename  string
	Kind      string
	Size      int64
	CreatedAt int64
}

func (s *Service) Archives(ctx context.Context) ([]Archive, error) {
	entries, err := os.ReadDir(s.Path)
	if os.IsNotExist(err) {
		return []Archive{}, nil
	}
	if err != nil {
		return nil, err
	}
	archives := make([]Archive, 0, len(entries))
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		name := entry.Name()
		if !validArchiveName(name) {
			continue
		}
		info, statErr := os.Lstat(s.Path + string(os.PathSeparator) + name)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, statErr
		}
		if !info.Mode().IsRegular() {
			continue
		}
		archives = append(archives, Archive{
			ID:        archiveID(name),
			Filename:  name,
			Kind:      archiveKind(name),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Unix(),
		})
	}
	sort.Slice(archives, func(i, j int) bool {
		if archives[i].CreatedAt == archives[j].CreatedAt {
			return archives[i].Filename > archives[j].Filename
		}
		return archives[i].CreatedAt > archives[j].CreatedAt
	})
	return archives, nil
}

func (s *Service) FindArchive(ctx context.Context, id string) (Archive, error) {
	archives, err := s.Archives(ctx)
	if err != nil {
		return Archive{}, err
	}
	for _, archive := range archives {
		if archive.ID == id {
			return archive, nil
		}
	}
	return Archive{}, ErrNotFound
}

func archiveID(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])
}

func archiveKind(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "tally-auto-"):
		return "auto"
	case strings.HasPrefix(lower, "tally-manual-"):
		return "manual"
	default:
		return "imported"
	}
}
