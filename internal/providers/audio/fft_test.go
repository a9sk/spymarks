package audio

import (
	"encoding/binary"
	"math"
	"os"
	"testing"

	"github.com/a9sk/spymarks/internal/core"
)

func makeTestWAV(path string, frames int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rate := 16000
	channels := 1
	bitsPerSample := 16
	numSamples := frames*defaultHop + defaultFrameSize
	dataSize := numSamples * channels * bitsPerSample / 8
	byteRate := rate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	binary.Write(f, binary.LittleEndian, []byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	binary.Write(f, binary.LittleEndian, []byte("WAVE"))
	binary.Write(f, binary.LittleEndian, []byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(channels))
	binary.Write(f, binary.LittleEndian, uint32(rate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))
	binary.Write(f, binary.LittleEndian, []byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	for i := 0; i < numSamples; i++ {
		// White-noise-like signal for uniform spectral energy.
		v := int16((math.Sin(2*math.Pi*440*float64(i)/float64(rate)) + 0.5*math.Sin(2*math.Pi*880*float64(i)/float64(rate))) * 3000)
		binary.Write(f, binary.LittleEndian, v)
	}
	return nil
}

func TestFFTMarkCheck(t *testing.T) {
	path := "/tmp/spymarks-fft-test.wav"
	marked := "/tmp/spymarks-fft-marked.wav"
	if err := makeTestWAV(path, 2000); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	defer os.Remove(marked)

	seed, err := core.KeyDerive("fft-test-key")
	if err != nil {
		t.Fatal(err)
	}
	payload := core.PayloadFromMessage("audio-test")

	cfg := core.DefaultEmbedConfig()
	cfg.Strength = 48.0
	cfg.Threshold = 0.6

	p := NewFFTProvider()
	if err := p.Mark(path, marked, payload, seed, cfg); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	res, err := p.Check(marked, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !res.Detected {
		t.Fatalf("expected detection, got confidence %.2f, payload %s", res.Confidence, res.Payload)
	}
	if res.Payload != payload.Hex() {
		t.Fatalf("payload mismatch: want %s got %s", payload.Hex(), res.Payload)
	}

	// Original audio should not be detected.
	res2, err := p.Check(path, payload.Len(), seed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Detected {
		t.Fatalf("original audio falsely detected with confidence %.2f", res2.Confidence)
	}
}

func TestFFTRoundTripSpectrum(t *testing.T) {
	in := make([]float64, 64)
	for i := range in {
		in[i] = math.Sin(2 * math.Pi * float64(i) / 16)
	}
	spec := fftReal(in)
	out := ifftReal(spec)
	maxErr := 0.0
	for i := range in {
		err := math.Abs(in[i] - real(out[i]))
		if err > maxErr {
			maxErr = err
		}
	}
	if maxErr > 1e-9 {
		t.Fatalf("FFT round-trip error too large: %g", maxErr)
	}
}

func BenchmarkFFTSpectrum(b *testing.B) {
	in := make([]float64, defaultFrameSize)
	for i := range in {
		in[i] = math.Sin(2 * math.Pi * float64(i) / 64)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spec := fftReal(in)
		_ = ifftReal(spec)
	}
}
