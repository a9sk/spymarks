package text

import (
	"os"
	"strings"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
)

func makeTestText(path string) error {
	// Repeat a sentence with many synonym words so the payload fits.
	// Default repeat ECC expands 96 bits to 288 bits.
	parts := []string{
		"The quick brown fox jumps over the lazy dog",
		"A good start is to make fast progress and help everyone feel happy",
		"We know that small changes can show great results when we use the right words",
		"Tell a clear story from start to finish with no bad outcomes",
	}
	var chunks []string
	for i := 0; i < 30; i++ {
		for _, p := range parts {
			chunks = append(chunks, p)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(chunks, ". ")+"."), 0644)
}

func TestSynonymMarkCheck(t *testing.T) {
	path := "/tmp/spymarks-text-test.txt"
	marked := "/tmp/spymarks-text-marked.txt"
	if err := makeTestText(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	seed, err := core.KeyDerive("text-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("text-test")

	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 1.0
	cfg.Threshold = 0.6

	p := NewSynonymProvider()
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

	// Original text should not be detected.
	res2, err := p.Check(path, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Detected {
		t.Fatalf("original text falsely detected with confidence %.2f", res2.Confidence)
	}
}

func TestSynonymCandidates(t *testing.T) {
	tokens := tokenize("A big happy start to make fast progress.")
	cands := candidatePositions(tokens)
	if len(cands) == 0 {
		t.Fatal("expected synonym candidates")
	}
}

func TestMatchCase(t *testing.T) {
	if got := matchCase("Happy", "glad"); got != "Glad" {
		t.Fatalf("expected Glad, got %s", got)
	}
	if got := matchCase("HAPPY", "glad"); got != "GLAD" {
		t.Fatalf("expected GLAD, got %s", got)
	}
	if got := matchCase("happy", "glad"); got != "glad" {
		t.Fatalf("expected glad, got %s", got)
	}
}
