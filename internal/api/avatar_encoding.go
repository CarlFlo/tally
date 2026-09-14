package api

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func builtinAvatar(v string) bool {
	return v == "violet" || v == "mint" || v == "amber" || v == "rose" || v == "blue" || v == "peach"
}

func ReencodeAvatar(data []byte) ([]byte, error) {
	if len(data) > 4<<20 {
		return nil, fmt.Errorf("avatar must be under 4 MB")
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
	if e != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return nil, fmt.Errorf("upload a JPEG, PNG, or WebP image")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width*cfg.Height > 16000000 {
		return nil, fmt.Errorf("avatar dimensions must be at most 4096 × 4096")
	}
	src, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return nil, fmt.Errorf("image could not be decoded")
	}
	bounds := src.Bounds()
	side := bounds.Dx()
	if bounds.Dy() < side {
		side = bounds.Dy()
	}
	crop := image.Rect(bounds.Min.X+(bounds.Dx()-side)/2, bounds.Min.Y+(bounds.Dy()-side)/2, bounds.Min.X+(bounds.Dx()+side)/2, bounds.Min.Y+(bounds.Dy()+side)/2)
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, draw.Src, nil)
	var out bytes.Buffer
	e = png.Encode(&out, dst)
	return out.Bytes(), e
}
