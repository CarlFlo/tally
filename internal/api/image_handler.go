package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

var errInvalidProviderImage = errors.New("provider returned an invalid image")

func validateImageFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	cfg, format, err := image.DecodeConfig(file)
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") || cfg.Width > 5000 || cfg.Height > 5000 || int64(cfg.Width)*int64(cfg.Height) > 20000000 {
		return errInvalidProviderImage
	}
	return nil
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
	download, err := s.Control.DownloadToTempFile(r.Context(), providers.Request{
		Provider: "tvmaze-images",
		URL:      raw,
		Trigger:  "image_cache",
		MaxBytes: 4 << 20,
	}, dir)
	if err != nil {
		return remote(err)
	}
	defer os.Remove(download.Path)
	if err = validateImageFile(download.Path); err != nil {
		if errors.Is(err, errInvalidProviderImage) {
			return bad(err.Error())
		}
		return err
	}
	if err = os.Rename(download.Path, path); err != nil {
		// Another concurrent request may have populated the same cache entry.
		if serveCachedImage(w, r, path) {
			return nil
		}
		return err
	}
	if !serveCachedImage(w, r, path) {
		return errors.New("cached image could not be opened")
	}
	return nil
}
