package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/providers"
)

func serveCachedImage(w http.ResponseWriter, r *http.Request, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	var header [512]byte
	n, _ := file.Read(header[:])
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return false
	}
	w.Header().Set("Content-Type", http.DetectContentType(header[:n]))
	w.Header().Set("Cache-Control", "private,max-age=604800")
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
	return true
}

func writeCachedImage(dir, path string, data []byte) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".image-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err == nil {
		err = file.Close()
	} else {
		_ = file.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func (s *Server) image(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	raw := r.URL.Query().Get("url")
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "static.tvmaze.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/uploads/images/") {
		return bad("image source is not allowed")
	}
	hash := sha256.Sum256([]byte(raw))
	dir := filepath.Join(s.Config.DataDir, "cache", "images")
	name := hex.EncodeToString(hash[:])
	path := filepath.Join(dir, name)
	if serveCachedImage(w, r, path) {
		return nil
	}
	res, err := s.Control.Do(r.Context(), providers.Request{Provider: "tvmaze-images", URL: raw, Trigger: "image_cache", Coalesce: true, MaxBytes: 4 << 20})
	if err != nil {
		return remote(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(res.Body))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") || cfg.Width > 5000 || cfg.Height > 5000 || cfg.Width*cfg.Height > 20000000 {
		return bad("provider returned an invalid image")
	}
	if err = writeCachedImage(dir, path, res.Body); err != nil {
		return err
	}
	w.Header().Set("Content-Type", http.DetectContentType(res.Body))
	w.Header().Set("Cache-Control", "private,max-age=604800")
	_, _ = w.Write(res.Body)
	return nil
}
