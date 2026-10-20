package image

import (
	"os"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
)

func TestLSBMarkCheck(t *testing.T) {
	path := "/tmp/spymarks-lsb-test.png"
	marked := "/tmp/spymarks-lsb-marked.png"
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
	cfg.Strength = 1.0
	cfg.Threshold = 0.6

	p := NewLSBProvider()
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

func TestLSBChannelSelection(t *testing.T) {
	path := "/tmp/spymarks-lsb-channel-src.png"
	marked := "/tmp/spymarks-lsb-channel-marked.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	seed, err := core.KeyDerive("lsb-channel-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("lsb-channel-test")

	cfg := core.DefaultEmbedConfig()
	cfg.Channel = 'b'

	p := NewLSBProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	res, err := p.Check(marked, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected {
		t.Fatalf("expected detection on blue channel, got confidence %.2f", res.Confidence)
	}
}
