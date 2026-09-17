package api

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/CarlFlo/tally/internal/torrent"
)

type selection struct {
	Profile string
	Data    []byte
	Expires time.Time
}

type torrentSelection struct {
	Result      torrent.SearchResult       `json:"result"`
	Target      *torrent.EpisodeTarget      `json:"target,omitempty"`
	Preliminary *torrent.ReleaseAssessment `json:"preliminary,omitempty"`
}

func encodeTorrentSelection(value torrentSelection) ([]byte, error) {
	return json.Marshal(value)
}

func decodeTorrentSelection(data []byte) (torrentSelection, error) {
	// New selections carry the authoritative episode target and preliminary
	// assessment. Keep a legacy fallback for short-lived selections/tests that
	// were created before the envelope existed.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return torrentSelection{}, err
	}
	if _, ok := probe["result"]; ok {
		var value torrentSelection
		if err := json.Unmarshal(data, &value); err != nil {
			return torrentSelection{}, err
		}
		return value, nil
	}
	var result torrent.SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return torrentSelection{}, err
	}
	return torrentSelection{Result: result}, nil
}

type selectionStore struct{ sync.Map }

func (s *selectionStore) prune() {
	count := 0
	s.Range(func(k, v any) bool {
		count++
		if time.Now().After(v.(selection).Expires) || count > 10000 {
			s.Delete(k)
		}
		return true
	})
}
