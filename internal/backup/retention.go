package backup

import (
	"context"
	"os"
	"sort"
	"strings"
)

func (s *Service) Retain(ctx context.Context) error {
	keep, err := s.retentionCount(ctx)
	if err != nil {
		return err
	}
	archives, err := s.Archives(ctx)
	if err != nil {
		return err
	}
	auto := make([]Archive, 0, len(archives))
	for _, archive := range archives {
		if archive.Kind == "auto" {
			auto = append(auto, archive)
		}
	}
	for _, archive := range auto[minimum(keep, len(auto)):] {
		path, pathErr := s.archivePath(archive.Filename)
		if pathErr != nil {
			if os.IsNotExist(pathErr) {
				continue
			}
			return pathErr
		}
		if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ListFiles(path string) ([]string, error) {
	entries, e := os.ReadDir(path)
	if e != nil {
		return nil, e
	}
	out := []string{}
	for _, v := range entries {
		if strings.HasSuffix(strings.ToLower(v.Name()), ".zip") {
			out = append(out, v.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
