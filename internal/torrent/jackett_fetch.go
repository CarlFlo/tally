package torrent

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/providers"
)

func (j *Jackett) validTorrentURL(raw string) bool {
	target, err := url.Parse(raw)
	base, baseErr := url.Parse(j.BaseURL)
	if err != nil || baseErr != nil || target.Scheme != base.Scheme || target.Host != base.Host || target.User != nil || target.Fragment != "" {
		return false
	}
	basePath := strings.TrimRight(base.Path, "/")
	return basePath == "" || target.Path == basePath || strings.HasPrefix(target.Path, basePath+"/")
}

func (j *Jackett) FetchTorrent(ctx context.Context, raw string) ([]byte, error) {
	if !j.validTorrentURL(raw) {
		return nil, fmt.Errorf("torrent URL is outside the configured Jackett instance")
	}
	response, err := j.Control.Do(ctx, providers.Request{Provider: j.ID(), URL: raw, Trigger: "torrent_send", MaxBytes: 8 << 20, NoRetry: true})
	if err != nil {
		return nil, err
	}
	if len(response.Body) < 2 || response.Body[0] != 'd' || response.Body[len(response.Body)-1] != 'e' {
		return nil, fmt.Errorf("Jackett did not return a torrent file")
	}
	return response.Body, nil
}
