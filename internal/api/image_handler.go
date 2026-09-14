package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/providers"
)

func (s *Server) image(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	raw := r.URL.Query().Get("url")
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "static.tvmaze.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/uploads/images/") {
		return bad("image source is not allowed")
	}
	hash := sha256.Sum256([]byte(raw))
	dir := filepath.Join(s.Config.DataDir, "cache", "images")
	name := hex.EncodeToString(hash[:])
	path := filepath.Join(dir, name)
	if data, e := os.ReadFile(path); e == nil {
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Header().Set("Cache-Control", "private,max-age=604800")
		_, _ = w.Write(data)
		return nil
	}
	res, e := s.Control.Do(r.Context(), providers.Request{Provider: "tvmaze-images", URL: raw, Trigger: "image_cache", TTL: 7 * 24 * time.Hour, MaxBytes: 4 << 20})
	if e != nil {
		return remote(e)
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(res.Body))
	if e != nil || (format != "png" && format != "jpeg" && format != "webp") || cfg.Width > 5000 || cfg.Height > 5000 || cfg.Width*cfg.Height > 20000000 {
		return bad("provider returned an invalid image")
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	if e = os.WriteFile(path, res.Body, 0600); e != nil {
		return e
	}
	w.Header().Set("Content-Type", http.DetectContentType(res.Body))
	w.Header().Set("Cache-Control", "private,max-age=604800")
	_, _ = w.Write(res.Body)
	return nil
}
