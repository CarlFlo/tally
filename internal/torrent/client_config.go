package torrent

import (
	"fmt"
)

type ClientConfig struct {
	Adapter  string            `json:"adapter"`
	Fields   map[string]string `json:"-"` // Raw fields may contain credentials. Never serialize them to an API response.
	Revision int64             `json:"revision"`
}

type ClientUpdate struct {
	Adapter  string            `json:"adapter"`
	Fields   map[string]string `json:"fields"`
	Revision int64             `json:"revision"`
}

type ClientView struct {
	Adapter           string            `json:"adapter"`
	Fields            map[string]string `json:"fields"`
	SecretsConfigured map[string]bool   `json:"secrets_configured"`
	Revision          int64             `json:"revision"`
}

func (c ClientConfig) validate() error {
	adapter := findClient(c.Adapter)
	if adapter == nil {
		return ErrNoClient
	}
	for _, field := range adapter.Fields {
		if field.Required && c.Fields[field.Key] == "" {
			return fmt.Errorf("%s is required; update the torrent client in Settings", field.Label)
		}
	}
	if adapter.validate != nil {
		return adapter.validate(c.Fields)
	}
	return nil
}

func (c ClientConfig) Configured() bool { return c.validate() == nil }

func (c ClientConfig) View() ClientView {
	view := ClientView{Adapter: c.Adapter, Fields: map[string]string{}, SecretsConfigured: map[string]bool{}, Revision: c.Revision}
	if adapter := findClient(c.Adapter); adapter != nil {
		for _, field := range adapter.Fields {
			if field.Secret {
				view.SecretsConfigured[field.Key] = c.Fields[field.Key] != ""
			} else {
				view.Fields[field.Key] = c.Fields[field.Key]
			}
		}
	}
	return view
}
