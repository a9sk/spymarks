package image

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
)

func TestDCTMarkCheckTestdata(t *testing.T) {
	seed, err := core.KeyDerive("testdata-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("testdata-test")
	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 6.0
	cfg.Threshold = 0.6
	cfg.Iterations = 2

	p := NewDCTProvider()

	files, err := filepath.Glob("../../../testdata/*.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no png testdata found")
	}

	for _, path := range files {
		base := filepath.Base(path)
		marked := filepath.Join(os.TempDir(), "spymarks-"+base+".marked.png")
		if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
			t.Fatalf("mark failed for %s: %v", path, err)
		}
		res, err := p.Check(marked, payload.Len(), seed, cfg)
		if err != nil {
			t.Fatalf("check failed for %s: %v", path, err)
		}
		if !res.Detected || res.Payload != payload.Hex() {
			t.Fatalf("%s: expected detection with payload %s, got detected=%v payload=%s confidence=%.2f", path, payload.Hex(), res.Detected, res.Payload, res.Confidence)
		}
		os.Remove(marked)
	}
}

func TestDCTSurvivesJPEGQuality70(t *testing.T) {
	path := "../../../testdata/sample.png"
	marked := "/tmp/spymarks-q70-marked.png"
	jpegPath := "/tmp/spymarks-q70.jpg"

	seed, err := core.KeyDerive("q70-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("q70-test")
	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 12.0
	cfg.Threshold = 0.6

	p := NewDCTProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(marked)
	defer os.Remove(jpegPath)

	if err := encodeJPEG(marked, jpegPath, 70); err != nil {
		t.Fatalf("jpeg encode failed: %v", err)
	}

	res, err := p.Check(jpegPath, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected || res.Payload != payload.Hex() {
		t.Fatalf("expected detection after JPEG quality 70, got detected=%v payload=%s confidence=%.2f", res.Detected, res.Payload, res.Confidence)
	}
}
