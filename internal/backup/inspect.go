package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const maxManifestSize = 64 << 10

func (s *Service) Inspect(ctx context.Context, filename string) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.archivePath(filename)
	if err != nil {
		return Manifest{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Manifest{}, err
	}
	if cached, ok := s.inspections[filename]; ok &&
		cached.size == info.Size() &&
		cached.modifiedAt == info.ModTime().UnixNano() {
		return cached.manifest, cached.err
	}
	manifest, inspectErr := inspectArchive(ctx, path)
	if !errors.Is(inspectErr, context.Canceled) && !errors.Is(inspectErr, context.DeadlineExceeded) {
		s.rememberInspection(filename, info, manifest, inspectErr)
	}
	return manifest, inspectErr
}

func (s *Service) rememberInspection(filename string, info os.FileInfo, manifest Manifest, err error) {
	if s.inspections == nil {
		s.inspections = make(map[string]inspectionCache)
	}
	s.inspections[filename] = inspectionCache{
		size:       info.Size(),
		modifiedAt: info.ModTime().UnixNano(),
		manifest:   manifest,
		err:        err,
	}
}

func inspectArchive(ctx context.Context, path string) (Manifest, error) {
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return Manifest{}, err
	}
	defer archive.Close()
	var manifestFile *zip.File
	for _, file := range archive.File {
		if file.Name != "manifest.json" {
			continue
		}
		if manifestFile != nil {
			return Manifest{}, fmt.Errorf("duplicate backup manifest")
		}
		manifestFile = file
	}
	if manifestFile == nil || manifestFile.UncompressedSize64 > maxManifestSize {
		return Manifest{}, fmt.Errorf("backup manifest is missing or too large")
	}
	reader, err := manifestFile.Open()
	if err != nil {
		return Manifest{}, err
	}
	defer reader.Close()
	var manifest Manifest
	if err = json.NewDecoder(io.LimitReader(reader, maxManifestSize+1)).Decode(&manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.Format != 1 || manifest.Schema < 1 || len(manifest.AppVersion) > 64 {
		return Manifest{}, fmt.Errorf("unsupported backup manifest")
	}
	return manifest, nil
}
