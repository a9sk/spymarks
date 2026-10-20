package image

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
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

func makeBlock(n int) [][]float64 {
	block := make([][]float64, n)
	for i := range block {
		block[i] = make([]float64, n)
	}
	return block
}

func TestDCTRoundTrip(t *testing.T) {
	n := 8
	block := makeBlock(n)
	for y := range block {
		for x := range block {
			block[y][x] = float64(x + y*n)
		}
	}

	d := dct2(block, n)
	recovered := idct2(d, n)

	maxErr := 0.0
	for y := range block {
		for x := range block {
			err := math.Abs(block[y][x] - recovered[y][x])
			if err > maxErr {
				maxErr = err
			}
		}
	}
	if maxErr > 1e-9 {
		t.Fatalf("DCT round-trip max error too large: %g", maxErr)
	}
}

func TestDCTCoefficientSign(t *testing.T) {
	n := 8
	block := makeBlock(n)
	for y := range block {
		for x := range block {
			block[y][x] = 128 + float64(x-y)*2
		}
	}

	d := dct2(block, n)
	if d[1][2] == 0 {
		t.Fatal("mid-frequency coefficient is zero")
	}

	d[1][2] += 5.0
	recovered := idct2(d, n)
	back := dct2(recovered, n)
	if back[1][2]*d[1][2] < 0 {
		t.Fatalf("sign flipped after round-trip: before %g after %g", d[1][2], back[1][2])
	}
}

func TestDCTMarkCheck(t *testing.T) {
	path := "/tmp/spymarks-dct-test.png"
	marked := "/tmp/spymarks-dct-marked.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	seed, err := core.KeyDerive("integration-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("user-1234")

	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 2.0
	cfg.Threshold = 0.6

	p := NewDCTProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	res, err := p.Check(marked, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected {
		t.Fatalf("expected detection, got confidence %.2f", res.Confidence)
	}
	if res.Payload != payload.Hex() {
		t.Fatalf("payload mismatch: want %s got %s", payload.Hex(), res.Payload)
	}

	// Original image should not be detected.
	res2, err := p.Check(path, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Detected {
		t.Fatalf("original image falsely detected with confidence %.2f", res2.Confidence)
	}
}

func TestDCTSurvivesJPEG(t *testing.T) {
	path := "/tmp/spymarks-dct-jpeg-src.png"
	marked := "/tmp/spymarks-dct-jpeg-marked.png"
	jpegPath := "/tmp/spymarks-dct-jpeg-resaved.jpg"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)
	defer os.Remove(jpegPath)

	seed, err := core.KeyDerive("jpeg-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("jpeg-test")

	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 4.0
	cfg.Threshold = 0.6

	p := NewDCTProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	if err := encodeJPEG(marked, jpegPath, 90); err != nil {
		t.Fatalf("jpeg encode failed: %v", err)
	}

	res, err := p.Check(jpegPath, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected {
		t.Fatalf("expected detection after JPEG re-encode, got confidence %.2f", res.Confidence)
	}
}

func TestDCTChannelSelection(t *testing.T) {
	testChannels := []byte{'r', 'g', 'b', 'c', 'C'}
	for _, ch := range testChannels {
		path := "/tmp/spymarks-dct-channel-src.png"
		marked := "/tmp/spymarks-dct-channel-marked.png"
		if err := makeTestImage(path); err != nil {
			t.Fatal(err)
		}

		seed, err := core.KeyDerive("channel-test-key")
		if err != nil {
			t.Fatal(err)
		}
		payload := core.PayloadFromMessage("channel-test")

		cfg := core.DefaultEmbedConfig()
		cfg.Strength = 4.0
		cfg.Channel = ch

		p := NewDCTProvider()
		if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
			os.Remove(path)
			t.Fatalf("mark failed for channel %c: %v", ch, err)
		}

		res, err := p.Check(marked, payload.Len(), seed, cfg)
		os.Remove(path)
		os.Remove(marked)
		if err != nil {
			t.Fatalf("check failed for channel %c: %v", ch, err)
		}
		if !res.Detected {
			t.Fatalf("expected detection on channel %c, got confidence %.2f", ch, res.Confidence)
		}
	}
}

func TestDCTIterations(t *testing.T) {
	path := "/tmp/spymarks-dct-iter-src.png"
	marked := "/tmp/spymarks-dct-iter-marked.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	seed, err := core.KeyDerive("iter-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("iter-test")

	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 2.0
	cfg.Iterations = 3

	p := NewDCTProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	res, err := p.Check(marked, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected {
		t.Fatalf("expected detection with iterations, got confidence %.2f", res.Confidence)
	}
}

func encodeJPEG(srcPath, dstPath string, quality int) error {
	img, _, err := Load(srcPath)
	if err != nil {
		return err
	}
	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return EncodeJPEG(f, img, quality)
}
