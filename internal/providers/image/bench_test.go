package image

import (
	"image"
	"image/color"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
)

func makeBenchImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	return img
}

func BenchmarkDCTMark256(b *testing.B) {
	img := makeBenchImage(256, 256)
	seed, _ := core.KeyDerive("bench-key")
	payload := core.PayloadFromMessage("bench")
	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 2.0
	p := NewDCTProvider()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp := image.NewRGBA(img.Bounds())
		copy(cp.Pix, img.Pix)
		_ = p.MarkImage(cp, payload, seed, cfg)
	}
}

func BenchmarkDCTCheck256(b *testing.B) {
	img := makeBenchImage(256, 256)
	seed, _ := core.KeyDerive("bench-key")
	payload := core.PayloadFromMessage("bench")
	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 2.0
	p := NewDCTProvider()
	cp := image.NewRGBA(img.Bounds())
	copy(cp.Pix, img.Pix)
	_ = p.MarkImage(cp, payload, seed, cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.CheckImage(cp, payload.Len(), seed, cfg)
	}
}

func BenchmarkLSBMark256(b *testing.B) {
	img := makeBenchImage(256, 256)
	seed, _ := core.KeyDerive("bench-key")
	payload := core.PayloadFromMessage("bench")
	cfg := core.DefaultEmbedConfig()
	p := NewLSBProvider()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp := image.NewRGBA(img.Bounds())
		copy(cp.Pix, img.Pix)
		_ = p.MarkImage(cp, payload, seed, cfg)
	}
}

func BenchmarkLSBCheck256(b *testing.B) {
	img := makeBenchImage(256, 256)
	seed, _ := core.KeyDerive("bench-key")
	payload := core.PayloadFromMessage("bench")
	cfg := core.DefaultEmbedConfig()
	p := NewLSBProvider()
	cp := image.NewRGBA(img.Bounds())
	copy(cp.Pix, img.Pix)
	_ = p.MarkImage(cp, payload, seed, cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.CheckImage(cp, payload.Len(), seed, cfg)
	}
}
