package torrent

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/providers"
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
	return q.Control.Do(ctx, providers.Request{Provider: "qbittorrent", URL: strings.TrimRight(q.URL, "/") + path, Method: method, Header: headers, Body: body, Trigger: trigger, NoRetry: true, MaxBytes: 1 << 20})
}
func (q *QBittorrent) TestConnection(ctx context.Context) error {
	res, e := q.call(ctx, "/api/v2/app/version", "GET", nil, "")
	if e != nil {
		return e
	}
	if !qBittorrentVersion.MatchString(strings.TrimSpace(string(res.Body))) {
		return fmt.Errorf("the server did not return a qBittorrent version; check the Web UI URL")
	}
	return nil
}
func (q *QBittorrent) AddMagnet(ctx context.Context, magnet string) error {
	if !ValidMagnet(magnet) {
		return fmt.Errorf("invalid magnet link")
	}
	v := url.Values{"urls": {magnet}}
	res, e := q.call(ctx, "/api/v2/torrents/add", "POST", []byte(v.Encode()), "application/x-www-form-urlencoded")
	if e == nil && strings.TrimSpace(string(res.Body)) != "Ok." {
		e = fmt.Errorf("qBittorrent did not accept the torrent")
	}
	return e
}
func (q *QBittorrent) AddTorrent(ctx context.Context, data []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, e := writer.CreateFormFile("torrents", "selected.torrent")
	if e != nil {
		return e
	}
	if _, e = part.Write(data); e != nil {
		return e
	}
	if e = writer.Close(); e != nil {
		return e
	}
	res, e := q.call(ctx, "/api/v2/torrents/add", "POST", body.Bytes(), writer.FormDataContentType())
	if e == nil && strings.TrimSpace(string(res.Body)) != "Ok." {
		e = fmt.Errorf("qBittorrent did not accept the torrent")
	}
	return e
}
