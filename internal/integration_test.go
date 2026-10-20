package internal

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

func TestEndToEndDCT(t *testing.T) {
	path := "/tmp/spymarks-e2e.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	opts := &Options{
		Key:      "e2e-test-key",
		Method:   "dct",
		Message:  "user-1234",
		Strength: 2.0,
	}

	res, err := Mark(path, opts)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(res.Output)

	checkOpts := &Options{
		Key:       "e2e-test-key",
		Method:    "dct",
		Message:   "user-1234",
		Strength:  2.0,
		Threshold: 0.6,
	}
	checkRes, err := Check(res.Output, checkOpts)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !checkRes.Detected {
		t.Fatalf("expected detection, got confidence %.2f", checkRes.Confidence)
	}
	if checkRes.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s check=%s", res.Payload, checkRes.Payload)
	}

	// Original image should not be detected.
	checkRes2, err := Check(path, checkOpts)
	if err != nil {
		t.Fatal(err)
	}
	if checkRes2.Detected {
		t.Fatalf("original image falsely detected with confidence %.2f", checkRes2.Confidence)
	}
}
