package api

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestAvatarValidation(t *testing.T) {
	if _, e := ReencodeAvatar([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)); e == nil {
		t.Fatal("SVG accepted")
	}
	var input bytes.Buffer
	_ = png.Encode(&input, image.NewRGBA(image.Rect(0, 0, 400, 200)))
	encoded, e := ReencodeAvatar(input.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	cfg, kind, e := image.DecodeConfig(bytes.NewReader(encoded))
	if e != nil || kind != "png" || cfg.Width != 256 || cfg.Height != 256 {
		t.Fatal("avatar not re-encoded to safe dimensions")
	}
	if _, e := ReencodeAvatar(make([]byte, (4<<20)+1)); e == nil {
		t.Fatal("oversize avatar accepted")
	}
}
