package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) torrentSend(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	enabled, e := s.torrentDownloadsEnabled(r.Context())
	if e != nil {
		return e
	}
	if !enabled {
		return bad("torrent downloads are disabled in Settings")
	}
	client, e := s.Clients.Current(r.Context())
	if e != nil {
		return clientInputError(e)
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
	payload, e := decodeTorrentSelection(selected.Data)
	if e != nil {
		return e
	}
	payload, e = s.resolveTorrentSelection(r.Context(), session.Profile, payload)
	if e != nil {
		return e
	}
	if encoded, encodeErr := encodeTorrentSelection(payload); encodeErr == nil {
		selected.Data = encoded
		s.selections.Store(in.Selection, selected)
	}
	result := payload.Result
	infoHash := result.InfoHash
	if infoHash == "" {
		infoHash = torrent.MagnetInfoHash(result.Magnet)
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

	inspectionRejected := false
	if result.Magnet != "" {
		// Magnet-only results cannot expose their payload before the client
		// retrieves BitTorrent metadata. They remain a manual-only path.
		e = client.AddMagnet(r.Context(), result.Magnet)
	} else {
		var data []byte
		provider, providerErr := s.searchProvider(r.Context(), "jackett")
		if providerErr != nil {
			e = providerErr
		} else if provider == nil {
			e = bad("the selected torrent provider is no longer configured")
		} else {
			data, e = provider.FetchTorrent(r.Context(), result.URL)
		}
		if e == nil {
			// Manual confidence is advisory. Re-evaluate the selected torrent's
			// actual payload from a clean assessment so a title/indexer metadata
			// concern cannot block an explicit user choice. Payload safety and
			// show/episode identity checks remain authoritative.
			base := torrent.ReleaseAssessment{Confidence: torrent.ConfidenceLow, Verification: torrent.VerificationUnverified}
			assessment, verifyErr := torrent.VerifyTorrentForTarget(base, result, data, payload.Target)
			e = verifyErr
			if assessment.InfoHash != "" {
				infoHash = assessment.InfoHash
			}
			inspectionRejected = e != nil
		}
		if e == nil {
			e = client.AddTorrent(r.Context(), data)
		}
	}
	status, errorText := "sent", ""
	if e != nil {
		status = "failed"
		errorText = e.Error()
		if !inspectionRejected {
			s.Jobs.Alert("downloader", "error", "Torrent submission failed. Check the client before retrying.")
		}
	}
	if _, writeErr := s.DB.Exec("UPDATE torrent_send_history SET status=?,error=? WHERE id=?", status, errorText, id); writeErr != nil {
		return writeErr
	}
	if e != nil {
		if inspectionRejected {
			return bad(e.Error())
		}
		return remote(e)
	}
	// Only an explicit library episode can be marked later. A target inferred
	// from free text may not belong to this deployment's library.
	if payload.EpisodeID != "" && infoHash != "" {
		if e = (torrent.EpisodeDownloadStore{DB: s.DB}).Track(r.Context(), infoHash, payload.EpisodeID); e != nil {
			return e
		}
	}
	jsonResponse(w, 200, map[string]string{"status": status})
	return nil
}
