package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

func writeArchive(ctx context.Context, path string, files map[string]string, manifest Manifest) error {
	file, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	zw := zip.NewWriter(file)
	for name, path := range files {
		if e = ctx.Err(); e != nil {
			zw.Close()
			file.Close()
			return e
		}
		source, e := os.Open(path)
		if e != nil {
			zw.Close()
			file.Close()
			return e
		}
		hash := sha256.New()
		w, e := zw.Create(name)
		if e == nil {
			_, e = io.Copy(io.MultiWriter(w, hash), &contextReader{ctx: ctx, reader: source})
		}
		source.Close()
		manifest.Files[name] = hex.EncodeToString(hash.Sum(nil))
		if e != nil {
			zw.Close()
			file.Close()
			return e
		}
	}
	w, e := zw.Create("manifest.json")
	if e == nil {
		e = json.NewEncoder(w).Encode(manifest)
	}
	closeErr := zw.Close()
	syncErr := file.Sync()
	fileErr := file.Close()
	for _, err := range []error{e, closeErr, syncErr, fileErr} {
		if err != nil {
			return err
		}
	}

	return nil
}
