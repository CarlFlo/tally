package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/CarlFlo/tally/internal/providers"
)

func qBittorrentAdapter() clientAdapter {
	return clientAdapter{
		ClientDefinition: ClientDefinition{
			ID: "qbittorrent", Name: "qBittorrent",
			Fields: []ClientField{
				{Key: "url", Label: "Web UI URL", Type: "url", Required: true, Placeholder: "http://qbittorrent:8080"},
				{Key: "api_key", Label: "API key", Type: "password", Required: true, Secret: true, Placeholder: "qbt_…", Help: "Generate an API key in qBittorrent Preferences → WebUI → API Key (qBittorrent 5.2+)."},
			},
		},
		create: func(control providers.Requester, fields map[string]string) DownloadClient {
			return &QBittorrent{Control: control, URL: fields["url"], APIKey: fields["api_key"]}
		},
		validate: func(fields map[string]string) error { return validateQBittorrentKey(fields["api_key"]) },
	}
}

var qBittorrentVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[a-zA-Z0-9.+_-]*)$`)
var qBittorrentKey = regexp.MustCompile(`^qbt_[A-Za-z0-9]+$`)
var qBittorrentHash = regexp.MustCompile(`^(?:[A-Fa-f0-9]{40}|[A-Fa-f0-9]{64})$`)

func validateQBittorrentKey(key string) error {
	if key == "" {
		return fmt.Errorf("enter the qBittorrent API key in Settings")
	}
	if !qBittorrentKey.MatchString(key) {
		return fmt.Errorf("enter the API key starting with qbt_, without Bearer or whitespace")
	}
	return nil
}

type QBittorrent struct {
	Control     providers.Requester
	URL, APIKey string
}

func (q *QBittorrent) Name() string { return "qBittorrent" }

func (q *QBittorrent) call(ctx context.Context, path, method string, body []byte, contentType string) (providers.Response, error) {
	if q.URL == "" {
		return providers.Response{}, fmt.Errorf("enter the qBittorrent Web UI URL in Settings")
	}
	if e := validateQBittorrentKey(q.APIKey); e != nil {
		return providers.Response{}, e
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+q.APIKey)
	headers.Set("Referer", strings.TrimRight(q.URL, "/")+"/")
	if base, e := url.Parse(q.URL); e == nil {
		headers.Set("Origin", base.Scheme+"://"+base.Host)
	}
	if contentType != "" {
		headers.Set("Content-Type", contentType)
	}
	trigger := "connection_test"
	if strings.Contains(path, "/torrents/") {
		trigger = "torrent_send"
	}
	return q.Control.Do(ctx, providers.Request{
		Provider: "qbittorrent",
		URL: strings.TrimRight(q.URL, "/") + path,
		Method: method,
		Header: headers,
		Body: body,
		Trigger: trigger,
		NoRetry: true,
		MaxBytes: 1 << 20,
	})
}

func (q *QBittorrent) TestConnection(ctx context.Context) error {
	res, e := q.call(ctx, "/api/v2/app/version", http.MethodGet, nil, "")
	if e != nil {
		return e
	}
	if !qBittorrentVersion.MatchString(strings.TrimSpace(string(res.Body))) {
		return fmt.Errorf("the server did not return a qBittorrent version; check the Web UI URL")
	}
	return nil
}

func (q *QBittorrent) ensureCategory(ctx context.Context, category string) error {
	res, e := q.call(ctx, "/api/v2/torrents/categories", http.MethodGet, nil, "")
	if e != nil {
		return e
	}
	var categories map[string]json.RawMessage
	if e := json.Unmarshal(res.Body, &categories); e != nil {
		return fmt.Errorf("qBittorrent returned invalid category data")
	}
	if _, ok := categories[category]; ok {
		return nil
	}
	values := url.Values{"category": {category}}
	if _, e = q.call(ctx, "/api/v2/torrents/createCategory", http.MethodPost, []byte(values.Encode()), "application/x-www-form-urlencoded"); e == nil {
		return nil
	}
	// Another request may have created the category between the list and create calls.
	res, retryErr := q.call(ctx, "/api/v2/torrents/categories", http.MethodGet, nil, "")
	if retryErr == nil && json.Unmarshal(res.Body, &categories) == nil {
		if _, ok := categories[category]; ok {
			return nil
		}
	}
	return e
}

func (q *QBittorrent) AddMagnet(ctx context.Context, magnet string) error {
	if !ValidMagnet(magnet) {
		return fmt.Errorf("invalid magnet link")
	}
	if e := q.ensureCategory(ctx, TallyCategory); e != nil {
		return e
	}
	values := url.Values{"urls": {magnet}, "category": {TallyCategory}}
	_, e := q.call(ctx, "/api/v2/torrents/add", http.MethodPost, []byte(values.Encode()), "application/x-www-form-urlencoded")
	return e
}

func (q *QBittorrent) AddTorrent(ctx context.Context, data []byte) error {
	if e := q.ensureCategory(ctx, TallyCategory); e != nil {
		return e
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, e := writer.CreateFormFile("torrents", "selected.torrent")
	if e != nil {
		return e
	}
	if _, e = part.Write(data); e != nil {
		return e
	}
	if e = writer.WriteField("category", TallyCategory); e != nil {
		return e
	}
	if e = writer.Close(); e != nil {
		return e
	}
	_, e = q.call(ctx, "/api/v2/torrents/add", http.MethodPost, body.Bytes(), writer.FormDataContentType())
	return e
}

type qBittorrentDownload struct {
	Hash       string  `json:"hash"`
	Name       string  `json:"name"`
	State      string  `json:"state"`
	Progress   float64 `json:"progress"`
	Size       int64   `json:"size"`
	Downloaded int64   `json:"downloaded"`
	DLSpeed    int64   `json:"dlspeed"`
	UPSpeed    int64   `json:"upspeed"`
	Ratio      float64 `json:"ratio"`
	AddedOn    int64   `json:"added_on"`
	Category   string  `json:"category"`
}

func (q *QBittorrent) Downloads(ctx context.Context, category string) (DownloadSnapshot, error) {
	values := url.Values{"category": {category}, "sort": {"added_on"}, "reverse": {"true"}}
	res, e := q.call(ctx, "/api/v2/torrents/info?"+values.Encode(), http.MethodGet, nil, "")
	if e != nil {
		return DownloadSnapshot{}, e
	}
	var rows []qBittorrentDownload
	if e := json.Unmarshal(res.Body, &rows); e != nil {
		return DownloadSnapshot{}, fmt.Errorf("qBittorrent returned invalid torrent data")
	}
	snapshot := DownloadSnapshot{Torrents: make([]Download, 0, len(rows))}
	for _, row := range rows {
		item := Download{
			Hash: row.Hash,
			Name: row.Name,
			State: row.State,
			Progress: row.Progress,
			Size: row.Size,
			Downloaded: row.Downloaded,
			DownloadSpeed: row.DLSpeed,
			UploadSpeed: row.UPSpeed,
			Ratio: row.Ratio,
			AddedOn: row.AddedOn,
			Category: row.Category,
		}
		snapshot.Torrents = append(snapshot.Torrents, item)
		snapshot.Stats.Total++
		snapshot.Stats.DownloadSpeed += row.DLSpeed
		snapshot.Stats.UploadSpeed += row.UPSpeed
		if row.DLSpeed > 0 || row.UPSpeed > 0 {
			snapshot.Stats.Active++
		}
	}
	return snapshot, nil
}


type qBittorrentFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (q *QBittorrent) ResolvedFiles(ctx context.Context, hash string) ([]TorrentFile, error) {
	if !qBittorrentHash.MatchString(hash) {
		return nil, fmt.Errorf("invalid torrent hash")
	}
	res, err := q.call(ctx, "/api/v2/torrents/files?"+url.Values{"hash": {hash}}.Encode(), http.MethodGet, nil, "")
	if err != nil {
		return nil, err
	}
	var rows []qBittorrentFile
	if err = json.Unmarshal(res.Body, &rows); err != nil {
		return nil, fmt.Errorf("qBittorrent returned invalid torrent file data")
	}
	if len(rows) > maxTorrentFiles {
		return nil, fmt.Errorf("qBittorrent returned too many torrent files")
	}
	files := make([]TorrentFile, 0, len(rows))
	for _, row := range rows {
		name := strings.ReplaceAll(strings.TrimSpace(row.Name), "\\", "/")
		if row.Size < 0 || len(name) == 0 || len(name) > 4096 {
			return nil, fmt.Errorf("qBittorrent returned invalid torrent file metadata")
		}
		segments := strings.Split(name, "/")
		for _, segment := range segments {
			if !validTorrentPathSegment(segment) {
				return nil, fmt.Errorf("qBittorrent returned an unsafe torrent file path")
			}
		}
		files = append(files, TorrentFile{Path: name, Size: row.Size})
	}
	return files, nil
}

func (q *QBittorrent) torrentAction(ctx context.Context, path, hash string, extra url.Values) error {
	if !qBittorrentHash.MatchString(hash) {
		return fmt.Errorf("invalid torrent hash")
	}
	values := url.Values{"hashes": {hash}}
	for key, list := range extra {
		for _, value := range list {
			values.Add(key, value)
		}
	}
	_, e := q.call(ctx, path, http.MethodPost, []byte(values.Encode()), "application/x-www-form-urlencoded")
	return e
}

func (q *QBittorrent) Stop(ctx context.Context, hash string) error {
	return q.torrentAction(ctx, "/api/v2/torrents/stop", hash, nil)
}

func (q *QBittorrent) Start(ctx context.Context, hash string) error {
	return q.torrentAction(ctx, "/api/v2/torrents/start", hash, nil)
}

func (q *QBittorrent) Remove(ctx context.Context, hash string, deleteFiles bool) error {
	return q.torrentAction(ctx, "/api/v2/torrents/delete", hash, url.Values{
		"deleteFiles": {fmt.Sprintf("%t", deleteFiles)},
	})
}
