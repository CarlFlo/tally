package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) uploadAvatar(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := r.ParseMultipartForm(4 << 20); e != nil {
		return bad("avatar upload is too large or invalid")
	}
	defer r.MultipartForm.RemoveAll()
	file, _, e := r.FormFile("avatar")
	if e != nil {
		return bad("choose an image to upload")
	}
	defer file.Close()
	data, e := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	if e != nil {
		return bad("image could not be read")
	}
	encoded, e := ReencodeAvatar(data)
	if e != nil {
		return bad(e.Error())
	}
	dir := filepath.Join(s.Config.DataDir, "avatars")
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	name := session.Profile + "-" + auth.Token()[:12] + ".png"
	if e = os.WriteFile(filepath.Join(dir, name), encoded, 0600); e != nil {
		return e
	}
	if _, e = s.DB.ExecContext(r.Context(), "UPDATE profiles SET avatar=? WHERE id=?", name, session.Profile); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]string{"avatar": name})
	return nil
}
