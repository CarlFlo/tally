package torrent

import "testing"

func TestMagnetInfoHashNormalizesBTIH(t *testing.T) {
	hexHash := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if got := MagnetInfoHash("magnet:?xt=urn:btih:" + hexHash); got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected hex infohash %q", got)
	}
	if got := MagnetInfoHash("magnet:?xt=urn:btih:VKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVK"); got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected base32 infohash %q", got)
	}
	if got := MagnetInfoHash("https://example.invalid/file"); got != "" {
		t.Fatalf("non-magnet produced an infohash: %q", got)
	}
}
