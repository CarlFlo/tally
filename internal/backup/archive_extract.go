package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func extractArchive(ctx context.Context, archive, dir string) (map[string]bool, error) {
	z, e := zip.OpenReader(archive)
	if e != nil {
		return nil, e
	}
	defer z.Close()
	if len(z.File) > 1000 {
		return nil, fmt.Errorf("too many archive entries")
	}
	seen := map[string]bool{}
	var total uint64
	for _, f := range z.File {
		if seen[f.Name] || !safeName(f.Name) || f.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe or duplicate archive entry")
		}
		seen[f.Name] = true
		total += f.UncompressedSize64
		if total > 2<<30 || f.UncompressedSize64 > 1<<30 {
			return nil, fmt.Errorf("archive exceeds size limit")
		}
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		rc, e := f.Open()
		if e != nil {
			return nil, e
		}
		target := filepath.Join(dir, filepath.FromSlash(f.Name))
		if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			rc.Close()
			return nil, e
		}
		out, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			rc.Close()
			return nil, e
		}
		_, e = io.Copy(out, io.LimitReader(rc, 1<<30+1))
		ce := out.Close()
		rc.Close()
		if e != nil {
			return nil, e
		}
		if ce != nil {
			return nil, ce
		}
	}

	return seen, nil
}
