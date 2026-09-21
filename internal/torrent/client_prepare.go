package torrent

import (
	"context"
	"net/url"
	"strings"

	"github.com/CarlFlo/tally/internal/config"
)

func (s *ClientStore) Prepare(ctx context.Context, in ClientUpdate) (ClientConfig, error) {
	saved, e := s.Load(ctx)
	if e != nil {
		return ClientConfig{}, e
	}
	if in.Revision != saved.Revision {
		return ClientConfig{}, ErrClientConflict
	}
	c := ClientConfig{Adapter: in.Adapter, Fields: map[string]string{}, Revision: saved.Revision}
	if in.Adapter == "" {
		if len(in.Fields) > 0 {
			return c, clientValidation("disabled clients cannot contain connection details")
		}
		return c, nil
	}
	adapter := findClient(in.Adapter)
	if adapter == nil {
		return c, clientValidation("choose a supported torrent client")
	}
	known := map[string]bool{}
	for _, field := range adapter.Fields {
		known[field.Key] = true
	}
	for key, value := range in.Fields {
		if !known[key] {
			return c, clientValidation("unknown torrent client field")
		}
		if len(value) > 4096 {
			return c, clientValidation("torrent client field is too long")
		}
		c.Fields[key] = value
	}
	for _, field := range adapter.Fields {
		if field.Secret {
			continue
		}
		value := strings.TrimSpace(c.Fields[field.Key])
		if field.Type == "url" && value != "" {
			if e := config.ValidateURL(value); e != nil {
				return c, clientValidation("%s must be an HTTP(S) address without embedded credentials", field.Label)
			}
			u, _ := url.Parse(value)
			if u.RawQuery != "" || u.ForceQuery {
				return c, clientValidation("%s cannot contain a query string", field.Label)
			}
			value = strings.TrimRight(value, "/")
		}
		c.Fields[field.Key] = value
	}
	for _, field := range adapter.Fields {
		if field.Secret {
			if _, supplied := in.Fields[field.Key]; !supplied && saved.Adapter == in.Adapter && saved.Fields[field.Key] != "" {
				c.Fields[field.Key] = saved.Fields[field.Key]
			}
		}
		if field.Required && c.Fields[field.Key] == "" {
			return c, clientValidation("%s is required", field.Label)
		}
	}
	if err := c.validate(); err != nil {
		return c, clientValidation("%s", err.Error())
	}
	return c, nil
}
