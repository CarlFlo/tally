package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) tallyDownloads(r *http.Request) (torrent.DownloadClient, torrent.DownloadSnapshot, error) {
	enabled, err := s.torrentDownloadsEnabled(r.Context())
	if err != nil {
		return nil, torrent.DownloadSnapshot{}, err
	}
	if !enabled {
		return nil, torrent.DownloadSnapshot{}, bad("torrent downloads are disabled in Settings")
	}
	client, e := s.Clients.Current(r.Context())
	if e != nil {
		return nil, torrent.DownloadSnapshot{}, clientInputError(e)
	}
	snapshot, e := client.Downloads(r.Context(), torrent.TallyCategory)
	if e != nil {
		return nil, torrent.DownloadSnapshot{}, remote(e)
	}
	return client, snapshot, nil
}

func (s *Server) torrentDownloads(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	_, snapshot, e := s.tallyDownloads(r)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, snapshot)
	return nil
}

func tallyTorrentExists(snapshot torrent.DownloadSnapshot, hash string) bool {
	for _, item := range snapshot.Torrents {
		if strings.EqualFold(item.Hash, hash) && item.Category == torrent.TallyCategory {
			return true
		}
	}
	return false
}

func (s *Server) torrentDownloadAction(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	client, snapshot, e := s.tallyDownloads(r)
	if e != nil {
		return e
	}
	hash := r.PathValue("hash")
	if !tallyTorrentExists(snapshot, hash) {
		return apiError{404, "Tally torrent not found"}
	}
	switch r.PathValue("action") {
	case "stop":
		e = client.Stop(r.Context(), hash)
	case "start":
		e = client.Start(r.Context(), hash)
	default:
		return apiError{404, "unknown torrent action"}
	}
	if e != nil {
		return remote(e)
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}

func (s *Server) removeTorrentDownload(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	client, snapshot, e := s.tallyDownloads(r)
	if e != nil {
		return e
	}
	hash := r.PathValue("hash")
	if !tallyTorrentExists(snapshot, hash) {
		return apiError{404, "Tally torrent not found"}
	}
	deleteFiles := false
	if raw := r.URL.Query().Get("delete_files"); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return bad("delete_files must be true or false")
		}
		deleteFiles = parsed
	}
	if e := client.Remove(r.Context(), hash, deleteFiles); e != nil {
		return remote(e)
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
