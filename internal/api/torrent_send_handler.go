package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/torrent"
)

func (s *Server) torrentSend(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	client, e := s.Clients.Current(r.Context())
	if e != nil {
		return bad(e.Error())
	}
	var in struct {
		Selection string `json:"selection"`
		Key       string `json:"idempotency_key"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if len(in.Key) < 16 || len(in.Key) > 100 {
		return bad("a valid idempotency key is required")
	}
	value, ok := s.selections.Load(in.Selection)
	if !ok {
		return bad("search result expired; search again")
	}
	selected := value.(selection)
	if selected.Profile != session.Profile || time.Now().After(selected.Expires) {
		return bad("search result expired; search again")
	}
	var result torrent.SearchResult
	if e := json.Unmarshal(selected.Data, &result); e != nil {
		return e
	}
	hash := auth.Digest(result.Magnet + "\x00" + result.URL)
	id := database.ID()
	res, e := s.DB.ExecContext(r.Context(), "INSERT INTO torrent_send_history VALUES(?,?,?,?,?,'pending','',?) ON CONFLICT(profile_id,idempotency_key) DO NOTHING", id, session.Profile, in.Key, hash, result.Name, time.Now().Unix())
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var oldHash, status string
		if e = s.DB.QueryRowContext(r.Context(), "SELECT request_hash,status FROM torrent_send_history WHERE profile_id=? AND idempotency_key=?", session.Profile, in.Key).Scan(&oldHash, &status); e != nil {
			return e
		}
		if oldHash != hash {
			return apiError{409, "this submission key belongs to another torrent"}
		}
		if status != "sent" {
			return apiError{409, "this send was already attempted; check the torrent client before trying again"}
		}
		jsonResponse(w, 200, map[string]string{"status": "sent"})
		return nil
	}
	if result.Magnet != "" {
		e = client.AddMagnet(r.Context(), result.Magnet)
	} else {
		var data []byte
		provider := s.searchProvider("jackett")
		if provider == nil {
			e = bad("the selected torrent provider is no longer configured")
		} else {
			data, e = provider.FetchTorrent(r.Context(), result.URL)
		}
		if e == nil {
			e = client.AddTorrent(r.Context(), data)
		}
	}
	status, errorText := "sent", ""
	if e != nil {
		status = "failed"
		errorText = e.Error()
		s.Jobs.Alert("downloader", "error", "Torrent submission failed. Check the client before retrying.")
	}
	if _, writeErr := s.DB.Exec("UPDATE torrent_send_history SET status=?,error=? WHERE id=?", status, errorText, id); writeErr != nil {
		return writeErr
	}
	if e != nil {
		return remote(e)
	}
	jsonResponse(w, 200, map[string]string{"status": status})
	return nil
}
