package backup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func installRestoredAvatars(stage, dataDir string) ([]string, error) {
	sourceDir := filepath.Join(stage, "avatars")
	entries, err := os.ReadDir(sourceDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	targetDir := filepath.Join(dataDir, "avatars")
	if err = os.MkdirAll(targetDir, 0700); err != nil {
		return nil, err
	}
	var created []string
	cleanup := func() {
		for _, path := range created {
			_ = os.Remove(path)
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !safeName("avatars/"+name) {
			cleanup()
			return nil, fmt.Errorf("invalid restored avatar")
		}
		source := filepath.Join(sourceDir, name)
		target := filepath.Join(targetDir, name)
		sourceData, readErr := os.ReadFile(source)
		if readErr != nil {
			cleanup()
			return nil, readErr
		}
		targetData, readErr := os.ReadFile(target)
		if readErr == nil {
			if !bytes.Equal(sourceData, targetData) {
				cleanup()
				return nil, fmt.Errorf("avatar filename conflicts with current data")
			}
			continue
		}
		if !os.IsNotExist(readErr) {
			cleanup()
			return nil, readErr
		}
		temp, createErr := os.CreateTemp(targetDir, ".restore-avatar-")
		if createErr != nil {
			cleanup()
			return nil, createErr
		}
		tempName := temp.Name()
		if createErr = temp.Chmod(0600); createErr == nil {
			_, createErr = temp.Write(sourceData)
		}
		if createErr == nil {
			createErr = temp.Sync()
		}
		closeErr := temp.Close()
		if createErr == nil {
			createErr = closeErr
		}
		if createErr == nil {
			createErr = os.Rename(tempName, target)
		}
		_ = os.Remove(tempName)
		if createErr != nil {
			cleanup()
			return nil, createErr
		}
		created = append(created, target)
	}
	return created, nil
}
