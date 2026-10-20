package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"strings"

	"github.com/a9sk/spymarks/internal/core"
)

const (
	defaultFrameSize = 1024
	defaultHop       = 1024
	defaultBandCount = 15
)

// FFTProvider embeds marks using FFT patchwork over overlapping audio frames.
type FFTProvider struct{}

// NewFFTProvider returns an FFT audio provider.
func NewFFTProvider() *FFTProvider {
	return &FFTProvider{}
}

func (p *FFTProvider) Method() string { return "fft" }

func (p *FFTProvider) Supported(mime, ext string) bool {
	ext = strings.ToLower(ext)
	return mime == "audio/wav" || mime == "audio/flac" || ext == ".wav" || ext == ".flac"
}

func (p *FFTProvider) Mark(inputPath, outputPath string, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
	samples, rate, err := loadAudio(inputPath)
	if err != nil {
		return err
	}

	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}
	bits := ecc.Encode(payload.WithSyncBits(syncBits))

	rng := core.NewDeterministicReader(seed)
	frameSize := defaultFrameSize
	hop := defaultHop
	numFrames := (len(samples) - frameSize) / hop
	if numFrames <= 0 {
		return fmt.Errorf("audio too short for watermark")
	}
	if len(bits) > numFrames {
		return fmt.Errorf("audio too short for payload: need %d frames, have %d", len(bits), numFrames)
	}

	// Center frequency band for embedding: avoid very low and very high bins.
	minBin := frameSize / 8
	maxBin := frameSize/2 - frameSize/8
	bandWidth := (maxBin - minBin) / (defaultBandCount * 2)
	if bandWidth < 1 {
		bandWidth = 1
	}

	frameOrder := rngPermute(rng, numFrames)

	modified := make([]float64, len(samples))
	copy(modified, samples)

	for i, bit := range bits {
		start := frameOrder[i] * hop

		frame := make([]float64, frameSize)
		copy(frame, samples[start:start+frameSize])

		spectrum := fftReal(frame)

		// Choose two disjoint bin sets deterministically.
		setA, setB := pickBinSets(rng, minBin, maxBin, bandWidth)

		sign := 1.0
		if rng.Bool() {
			sign = -1.0
		}
		delta := cfg.Strength * 0.01
		if !bit {
			delta = -delta
		}

		for _, b := range setA {
			spectrum[b] *= complex(1+sign*delta, 0)
			if b != 0 && b != frameSize/2 {
				spectrum[frameSize-b] *= complex(1+sign*delta, 0)
			}
		}
		for _, b := range setB {
			spectrum[b] *= complex(1-sign*delta, 0)
			if b != 0 && b != frameSize/2 {
				spectrum[frameSize-b] *= complex(1-sign*delta, 0)
			}
		}

		timeFrame := ifftReal(spectrum)
		for i := 0; i < frameSize; i++ {
			modified[start+i] = real(timeFrame[i])
		}
	}

	if cfg.DryRun {
		return nil
	}

	return saveAudio(outputPath, modified, rate, cfg.Lossless)
}

func (p *FFTProvider) Check(inputPath string, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	samples, _, err := loadAudio(inputPath)
	if err != nil {
		return nil, err
	}

	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return nil, err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}
	bitsToRead := int(float64(syncBits+payloadLen) * ecc.Expansion())

	rng := core.NewDeterministicReader(seed)
	frameSize := defaultFrameSize
	hop := defaultHop
	numFrames := (len(samples) - frameSize) / hop
	if numFrames <= 0 {
		return nil, fmt.Errorf("audio too short for watermark")
	}
	if bitsToRead > numFrames {
		return nil, fmt.Errorf("audio too short for payload")
	}

	minBin := frameSize / 8
	maxBin := frameSize/2 - frameSize/8
	bandWidth := (maxBin - minBin) / (defaultBandCount * 2)
	if bandWidth < 1 {
		bandWidth = 1
	}

	frameOrder := rngPermute(rng, numFrames)

	extracted := make([]bool, bitsToRead)

	for i := 0; i < bitsToRead; i++ {
		start := frameOrder[i] * hop

		frame := samples[start : start+frameSize]
		spectrum := fftReal(frame)

		setA, setB := pickBinSets(rng, minBin, maxBin, bandWidth)

		energyA := 0.0
		for _, b := range setA {
			energyA += cmplxAbs(spectrum[b])
		}
		energyB := 0.0
		for _, b := range setB {
			energyB += cmplxAbs(spectrum[b])
		}

		sign := 1.0
		if rng.Bool() {
			sign = -1.0
		}
		extracted[i] = sign*(energyA-energyB) > 0
	}

	decoded := ecc.Decode(extracted)
	if len(decoded) < syncBits {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: 0.0,
		}, nil
	}

	syncGot := core.PayloadFromBits(decoded[:syncBits]).Uint64()
	payloadBits := decoded[syncBits:]

	syncExpected := core.SyncPattern(syncBits) & ((1 << syncBits) - 1)
	matchingSync := syncBits - bits.OnesCount64(syncGot^syncExpected)
	confidence := float64(matchingSync) / float64(syncBits)

	if matchingSync < syncBits/2 {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: 0.0,
		}, nil
	}

	if confidence < cfg.Threshold {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: confidence,
		}, nil
	}

	payload := core.PayloadFromBits(payloadBits)
	return &core.CheckResult{
		Detected:   true,
		Method:     p.Method(),
		Payload:    payload.Hex(),
		Confidence: confidence,
	}, nil
}

func pickBinSets(rng *core.DeterministicReader, minBin, maxBin, width int) ([]int, []int) {
	setA := make([]int, defaultBandCount)
	setB := make([]int, defaultBandCount)

	// Split the available range in two to guarantee disjointness.
	mid := (minBin + maxBin) / 2
	startA := minBin + rng.IntN(max(1, mid-minBin-width*defaultBandCount))
	startB := mid + rng.IntN(max(1, maxBin-mid-width*defaultBandCount))

	for i := range setA {
		setA[i] = startA + i*width + rng.IntN(width)
		setB[i] = startB + i*width + rng.IntN(width)
	}
	return setA, setB
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func rngPermute(rng *core.DeterministicReader, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	for i := len(out) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func cmplxAbs(c complex128) float64 {
	return math.Hypot(real(c), imag(c))
}

// fftReal computes the forward FFT of a real signal (power-of-2 length).
func fftReal(in []float64) []complex128 {
	n := len(in)
	out := make([]complex128, n)
	for i, v := range in {
		out[i] = complex(v, 0)
	}
	fft(out, false)
	return out
}

// ifftReal computes the inverse FFT and returns real parts.
func ifftReal(in []complex128) []complex128 {
	out := make([]complex128, len(in))
	copy(out, in)
	fft(out, true)
	n := float64(len(out))
	for i := range out {
		out[i] /= complex(n, 0)
	}
	return out
}

// fft performs an in-place iterative Cooley-Tukey FFT.
func fft(a []complex128, inverse bool) {
	n := len(a)
	if n == 0 || n&(n-1) != 0 {
		panic("fft: length must be a power of two")
	}

	// Bit-reversal permutation.
	j := 0
	for i := 0; i < n; i++ {
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
	}

	sign := -1.0
	if inverse {
		sign = 1.0
	}

	for length := 2; length <= n; length <<= 1 {
		angle := 2 * math.Pi / float64(length) * sign
		wlen := complex(math.Cos(angle), math.Sin(angle))
		for i := 0; i < n; i += length {
			w := complex(1, 0)
			for k := 0; k < length/2; k++ {
				u := a[i+k]
				v := a[i+k+length/2] * w
				a[i+k] = u + v
				a[i+k+length/2] = u - v
				w *= wlen
			}
		}
	}
}

// Audio I/O.

type audioData struct {
	samples []float64
	rate    int
}

func loadAudio(path string) ([]float64, int, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".wav" {
		return loadWAV(path)
	}
	if ext == ".flac" {
		return nil, 0, fmt.Errorf("FLAC support not yet implemented")
	}
	return nil, 0, fmt.Errorf("unsupported audio format: %s", ext)
}

func saveAudio(path string, samples []float64, rate int, lossless bool) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".flac" || lossless {
		return saveWAV(path, samples, rate)
	}
	if ext == ".wav" {
		return saveWAV(path, samples, rate)
	}
	return fmt.Errorf("unsupported audio output format: %s", ext)
}

func loadWAV(path string) ([]float64, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var header [44]byte
	if _, err := f.Read(header[:]); err != nil {
		return nil, 0, fmt.Errorf("reading wav header: %w", err)
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}

	channels := int(binary.LittleEndian.Uint16(header[22:24]))
	rate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))
	dataSize := int(binary.LittleEndian.Uint32(header[40:44]))

	data := make([]byte, dataSize)
	if _, err := f.Read(data); err != nil {
		return nil, 0, fmt.Errorf("reading wav data: %w", err)
	}

	numSamples := dataSize / (bitsPerSample / 8)
	rawSamples := make([]int, numSamples)
	if bitsPerSample == 16 {
		for i := 0; i < numSamples; i++ {
			rawSamples[i] = int(int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2])))
		}
	} else if bitsPerSample == 24 {
		for i := 0; i < numSamples; i++ {
			b := data[i*3 : i*3+3]
			v := int(b[0]) | int(b[1])<<8 | int(b[2])<<16
			if v&0x800000 != 0 {
				v |= ^0xFFFFFF
			}
			rawSamples[i] = v
		}
	} else {
		return nil, 0, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
	}

	// Convert to mono float64 in [-1,1].
	frames := numSamples / channels
	samples := make([]float64, frames)
	scale := 1.0
	if bitsPerSample == 16 {
		scale = 32768.0
	} else if bitsPerSample == 24 {
		scale = 8388608.0
	}
	for i := 0; i < frames; i++ {
		sum := 0.0
		for ch := 0; ch < channels; ch++ {
			sum += float64(rawSamples[i*channels+ch]) / scale
		}
		samples[i] = sum / float64(channels)
	}
	return samples, rate, nil
}

func saveWAV(path string, samples []float64, rate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bitsPerSample := 16
	channels := 1
	byteRate := rate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(samples) * channels * bitsPerSample / 8

	// RIFF header.
	binary.Write(f, binary.LittleEndian, []byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	binary.Write(f, binary.LittleEndian, []byte("WAVE"))

	// fmt chunk.
	binary.Write(f, binary.LittleEndian, []byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(f, binary.LittleEndian, uint16(channels))
	binary.Write(f, binary.LittleEndian, uint32(rate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))

	// data chunk.
	binary.Write(f, binary.LittleEndian, []byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	for _, s := range samples {
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		v := int16(s * 32767)
		binary.Write(f, binary.LittleEndian, v)
	}
	return nil
}
