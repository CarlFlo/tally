package torrent

import "testing"

func TestClientDefinitionsDoNotShareMutableState(t *testing.T) {
	first := ClientDefinitions()
	first[0].Name = "Changed by a caller"
	first[0].Fields[0].Key = "unexpected"
	second := ClientDefinitions()
	if second[0].Name != "qBittorrent" || second[0].Fields[0].Key != "url" {
		t.Fatal("caller changed the compiled adapter definitions")
	}
	adapter := findClient("qbittorrent")
	if adapter == nil || adapter.Fields[0].Key != "url" {
		t.Fatal("caller changed adapter validation fields")
	}
}
