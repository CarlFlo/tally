package backup

import "context"

// Extract verifies every archive entry before the staged data may be restored.
func Extract(ctx context.Context, archive, dir string) (Manifest, error) {
	seen, err := extractArchive(ctx, archive, dir)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := readManifest(dir, seen)
	if err != nil {
		return manifest, err
	}
	return manifest, validateSnapshot(ctx, dir, manifest, seen)
}
