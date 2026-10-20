package watermark

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func makeTestImage(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func TestPublicAPIMarkDetect(t *testing.T) {
	path := "/tmp/spymarks-public-src.png"
	marked := "/tmp/spymarks-public-marked.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	cfg := Config{
		Key:      "public-api-key",
		Method:   "dct",
		Strength: 6.0,
	}
	res, err := Mark(path, marked, "public-test", cfg)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	det, err := Detect(marked, "public-test", cfg)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !det.Detected {
		t.Fatalf("expected detection, got confidence %.2f", det.Confidence)
	}
	if det.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s detect=%s", res.Payload, det.Payload)
	}
}
